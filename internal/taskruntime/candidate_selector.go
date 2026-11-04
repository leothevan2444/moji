package taskruntime

import (
	"context"
	"errors"
	"github.com/leothevan2444/moji/internal/indexer"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/mediafiles"
)

type torrentInspection struct {
	Paths       []string
	VideoPaths  []string
	SingleVideo bool
}

type torrentInspector func(ctx context.Context, torrentURL string) (torrentInspection, error)

type defaultCandidateSelector struct {
	inspectTorrent torrentInspector
}

func (s defaultCandidateSelector) Select(ctx context.Context, query string, results []indexer.SelectionCandidate, cfg config.CandidateSelectionConfig) (indexer.SelectionCandidate, error) {
	preview, err := s.Preview(ctx, query, results, cfg, true, true)
	if err != nil {
		return indexer.SelectionCandidate{}, err
	}
	for _, result := range preview.Results {
		if preferredTorrentURL(result) != "" {
			return result, nil
		}
	}
	return indexer.SelectionCandidate{}, errors.New("taskruntime: no downloadable torrent candidate found")
}

func (s defaultCandidateSelector) Preview(ctx context.Context, query string, results []indexer.SelectionCandidate, cfg config.CandidateSelectionConfig, applyFastRules bool, applyFileRules bool) (CandidateSelectionPreview, error) {
	if !cfg.Enabled {
		return CandidateSelectionPreview{Results: append([]indexer.SelectionCandidate(nil), results...)}, ctx.Err()
	}
	cfg = cfg.Effective()
	if !applyFastRules && !applyFileRules {
		return CandidateSelectionPreview{
			Results: append([]indexer.SelectionCandidate(nil), results...),
		}, nil
	}

	candidates := make([]rankedCandidate, 0, len(results))
	skipped := make([]rankedCandidate, 0, len(results))
	for index, result := range results {
		if preferredTorrentURL(result) == "" {
			skipped = append(skipped, rankedCandidate{
				index:  index,
				result: result,
				rank:   index,
			})
			continue
		}
		candidates = append(candidates, rankedCandidate{
			index:  index,
			result: result,
			rank:   index,
		})
	}
	if len(candidates) == 0 {
		out := make([]indexer.SelectionCandidate, 0, len(results))
		for _, skippedCandidate := range skipped {
			out = append(out, skippedCandidate.result)
		}
		return CandidateSelectionPreview{Results: out}, nil
	}

	compiled := compileSelectionRules(cfg.OrderedRules())
	fastRules, fileRules := splitCompiledRules(compiled)
	if applyFastRules {
		candidates = filterCandidates(candidates, func(c rankedCandidate) bool { return titleExclusionsAccept(c.result.Title, fastRules) })
	}
	if applyFastRules && len(fastRules) > 0 {
		sort.SliceStable(candidates, func(i, j int) bool {
			return compareRankedCandidates(query, candidates[i], candidates[j], fastRules) < 0
		})
	}
	for i := range candidates {
		candidates[i].rank = i
	}

	preview := CandidateSelectionPreview{
		Meta: CandidateSelectionPreviewMeta{
			AppliedFastRules: applyFastRules && len(fastRules) > 0,
			AppliedFileRules: applyFileRules && len(fileRules) > 0 && s.inspectTorrent != nil,
		},
	}
	if applyFileRules && hasFileExclusions(fileRules) && s.inspectTorrent == nil {
		candidates = nil
	}
	inspections := map[int]inspectedCandidate{}
	if preview.Meta.AppliedFileRules {
		inspectionLimit := config.NormalizeTorrentInspectionCandidateLimit(cfg.InspectionCandidateLimit)
		var inspectableCount int
		inspections, inspectableCount = inspectTopTorrentCandidates(ctx, candidates, s.inspectTorrent, inspectionLimit)
		preview.Meta.InspectableCount = inspectableCount
		preview.Meta.InspectedCount = len(inspections)
		if hasFileExclusions(fileRules) {
			candidates = filterCandidates(candidates, func(c rankedCandidate) bool { return fileExclusionsAccept(inspections[c.index], fileRules) })
		}
		limit := minInt(len(candidates), inspectionLimit)
		sort.SliceStable(candidates[:limit], func(i, j int) bool {
			return compareInspectedCandidates(candidates[i], candidates[j], fileRules, inspections) < 0
		})
		for i := range candidates {
			candidates[i].rank = i
		}
	}

	// Locking is OR across title and file clauses, after all exclusions. Keep
	// ordinary ranking within the locked set; no matches leave results intact.
	locked := make([]rankedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if applyFastRules && titleLockMatches(candidate.result.Title, fastRules) ||
			preview.Meta.AppliedFileRules && fileLockMatches(inspections[candidate.index], fileRules) {
			locked = append(locked, candidate)
		}
	}
	if len(locked) > 0 {
		candidates = locked
	}

	out := make([]indexer.SelectionCandidate, 0, len(results))
	for _, candidate := range candidates {
		out = append(out, candidate.result)
	}
	for _, skippedCandidate := range skipped {
		out = append(out, skippedCandidate.result)
	}
	preview.Results = out
	return preview, ctx.Err()
}

type rankedCandidate struct {
	index  int
	rank   int
	result indexer.SelectionCandidate
}

type compiledRule struct {
	rule          config.CandidateSelectionRule
	regexMatchers []*regexp.Regexp
}

type inspectedCandidate struct {
	inspection torrentInspection
	ok         bool
}

// A new preview must validate its source even when inspection bytes are cached.
// Automatic sourcing keeps using its already admitted search snapshot.
func (s *Service) inspectCurrentSearchResultTorrent(ctx context.Context, torrentURL string) (torrentInspection, error) {
	if strings.HasPrefix(torrentURL, "indexer-resource:") {
		if s.resourceAccess == nil {
			return torrentInspection{}, indexer.ErrNotConfigured
		}
		lease, err := s.resourceAccess.ResolveReference(ctx, strings.TrimPrefix(torrentURL, "indexer-resource:"))
		if err != nil {
			return torrentInspection{}, err
		}
		defer lease.Release()
		return indexedTorrentInspection(s.resolveIndexedArtifact(ctx, torrentURL, func(work context.Context) (indexer.Artifact, error) {
			return s.resourceAccess.FetchRecord(work, lease.Record)
		}))
	}
	return s.inspectSearchResultTorrent(ctx, torrentURL)
}

func indexedTorrentInspection(metadata downloadedTorrentMetadata, magnet string, err error) (torrentInspection, error) {
	if err != nil {
		return torrentInspection{}, err
	}
	if magnet != "" {
		return torrentInspection{}, errors.New("magnet resource has no inspectable torrent files")
	}
	return inspectTorrentMetadata(metadata), nil
}

func (s *Service) inspectSearchResultTorrent(ctx context.Context, torrentURL string) (torrentInspection, error) {
	if strings.HasPrefix(torrentURL, "indexer-resource:") {
		return indexedTorrentInspection(s.resolveIndexedResource(ctx, torrentURL))
	}
	if inspection, ok := s.cachedTorrentInspection(torrentURL); ok {
		return inspection, nil
	}
	metadata, err := s.fetchTorrentMetadata(ctx, torrentURL)
	if err != nil {
		return torrentInspection{}, err
	}
	inspection := inspectTorrentMetadata(metadata)
	s.storeCachedArtifact(torrentURL, &metadata, "")
	return inspection, nil
}

func (s *Service) cachedTorrentInspection(torrentURL string) (torrentInspection, bool) {
	key := normalizeTorrentInspectionCacheKey(torrentURL, "")
	if key == "" {
		return torrentInspection{}, false
	}
	now := s.now().UTC()
	s.inspectionCacheMu.Lock()
	defer s.inspectionCacheMu.Unlock()
	entry, ok := s.inspectionCache[key]
	if !ok {
		return torrentInspection{}, false
	}
	if !entry.expiresAt.After(now) {
		delete(s.inspectionCache, key)
		return torrentInspection{}, false
	}
	return entry.inspection, true
}

func (s *Service) storeCachedTorrentInspection(torrentURL string, inspection torrentInspection) {
	key := normalizeTorrentInspectionCacheKey(torrentURL, "")
	if key == "" {
		return
	}
	s.storeInspectionEntry(torrentURL, cachedTorrentInspection{
		inspection: inspection,
		expiresAt:  s.now().UTC().Add(torrentInspectionCacheTTL),
	})
}

func normalizeTorrentInspectionCacheKey(torrentURL string, infoHash string) string {
	if trimmed := strings.TrimSpace(infoHash); trimmed != "" {
		return "hash:" + strings.ToUpper(trimmed)
	}
	if trimmed := strings.TrimSpace(torrentURL); trimmed != "" {
		return "url:" + trimmed
	}
	return ""
}

func inspectTorrentMetadata(metadata downloadedTorrentMetadata) torrentInspection {
	paths := append([]string(nil), metadata.Paths...)
	if len(paths) == 0 && metadata.FilePath != "" {
		paths = append(paths, metadata.FilePath)
	}
	videoPaths := make([]string, 0, len(paths))
	for _, item := range paths {
		if isVideoFilePath(item) {
			videoPaths = append(videoPaths, item)
		}
	}
	return torrentInspection{
		Paths:       paths,
		VideoPaths:  videoPaths,
		SingleVideo: len(videoPaths) == 1,
	}
}

func compileSelectionRules(rules []config.CandidateSelectionRule) []compiledRule {
	out := make([]compiledRule, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled || rule.Type == config.CandidateSelectionRuleTypeIndexerPreference {
			continue
		}
		item := compiledRule{rule: rule}
		switch rule.Type {
		case config.CandidateSelectionRuleTypeTitleMatch:
			item.regexMatchers = compileRegexMatchers(rule.TitleMatch.Clauses)
		case config.CandidateSelectionRuleTypeTorrentFileNameMatch:
			item.regexMatchers = compileTorrentFileRegexMatchers(rule.TorrentFileNameMatch.Clauses)
		}
		out = append(out, item)
	}
	return out
}

func compileRegexMatchers(clauses []config.TitleMatchClause) []*regexp.Regexp {
	matchers := make([]*regexp.Regexp, len(clauses))
	for i, clause := range clauses {
		if clause.PatternMode != config.TitleMatchPatternModeRegex {
			continue
		}
		re, err := regexp.Compile(clause.Pattern)
		if err != nil {
			continue
		}
		matchers[i] = re
	}
	return matchers
}

func compileTorrentFileRegexMatchers(clauses []config.TorrentFileNameMatchClause) []*regexp.Regexp {
	matchers := make([]*regexp.Regexp, len(clauses))
	for i, clause := range clauses {
		if clause.PatternMode != config.TitleMatchPatternModeRegex {
			continue
		}
		re, err := regexp.Compile(clause.Pattern)
		if err != nil {
			continue
		}
		matchers[i] = re
	}
	return matchers
}

func splitCompiledRules(rules []compiledRule) (fast []compiledRule, file []compiledRule) {
	fast = make([]compiledRule, 0, len(rules))
	file = make([]compiledRule, 0, len(rules))
	for _, rule := range rules {
		switch rule.rule.Type {
		case config.CandidateSelectionRuleTypeTorrentSingleVideo,
			config.CandidateSelectionRuleTypeTorrentFileNameMatch:
			file = append(file, rule)
		default:
			fast = append(fast, rule)
		}
	}
	return fast, file
}

func compareRankedCandidates(query string, left rankedCandidate, right rankedCandidate, rules []compiledRule) int {
	for _, rule := range rules {
		if cmp := compareByRule(query, left.result, right.result, rule); cmp != 0 {
			return cmp
		}
	}
	if left.rank < right.rank {
		return -1
	}
	if left.rank > right.rank {
		return 1
	}
	return 0
}

func compareInspectedCandidates(left rankedCandidate, right rankedCandidate, rules []compiledRule, inspections map[int]inspectedCandidate) int {
	leftInspection := inspections[left.index]
	rightInspection := inspections[right.index]
	for _, rule := range rules {
		if cmp := compareByInspectionRule(leftInspection, rightInspection, rule); cmp != 0 {
			return cmp
		}
	}
	if left.rank < right.rank {
		return -1
	}
	if left.rank > right.rank {
		return 1
	}
	return 0
}

func compareByRule(query string, left indexer.SelectionCandidate, right indexer.SelectionCandidate, rule compiledRule) int {
	switch rule.rule.Type {
	case config.CandidateSelectionRuleTypeIndexerPreference:
		return compareInts(indexerPreferenceRank(left, rule.rule), indexerPreferenceRank(right, rule.rule), config.CandidateSelectionDirectionAsc)
	case config.CandidateSelectionRuleTypeTitleMatch:
		return compareInts(titleMatchRank(left.Title, rule), titleMatchRank(right.Title, rule), config.CandidateSelectionDirectionAsc)
	case config.CandidateSelectionRuleTypePublishDate:
		leftTime, leftOK := parsePublishDate(left.PublishDate)
		rightTime, rightOK := parsePublishDate(right.PublishDate)
		return compareTimes(leftTime, leftOK, rightTime, rightOK, rule.rule.PublishDate.Direction)
	case config.CandidateSelectionRuleTypeTitleSimilarity:
		return compareInts(titleSimilarityScore(query, left.Title), titleSimilarityScore(query, right.Title), config.CandidateSelectionDirectionDesc)
	case config.CandidateSelectionRuleTypeSeeders:
		return compareInts(left.Seeders, right.Seeders, rule.rule.Seeders.Direction)
	case config.CandidateSelectionRuleTypeSize:
		return compareInt64s(left.Size, right.Size, rule.rule.Size.Direction)
	default:
		return 0
	}
}

func compareByInspectionRule(left inspectedCandidate, right inspectedCandidate, rule compiledRule) int {
	switch rule.rule.Type {
	case config.CandidateSelectionRuleTypeTorrentSingleVideo:
		return compareInts(singleVideoRank(left), singleVideoRank(right), config.CandidateSelectionDirectionAsc)
	case config.CandidateSelectionRuleTypeTorrentFileNameMatch:
		leftRank := torrentFileNameMatchRank(left, rule)
		rightRank := torrentFileNameMatchRank(right, rule)
		return compareInts(leftRank, rightRank, config.CandidateSelectionDirectionAsc)
	default:
		return 0
	}
}

func inspectTopTorrentCandidates(ctx context.Context, candidates []rankedCandidate, inspector torrentInspector, limit int) (map[int]inspectedCandidate, int) {
	out := make(map[int]inspectedCandidate, limit)
	limit = minInt(len(candidates), limit)
	inspectableCount := 0
	var (
		outMu sync.Mutex
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 2)
	)
	for i := 0; i < limit; i++ {
		candidate := candidates[i]
		torrentURL := inspectableTorrentURL(candidate.result)
		if torrentURL == "" {
			continue
		}
		inspectableCount++
		wg.Add(1)
		go func(candidate rankedCandidate, torrentURL string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			inspection, err := inspector(ctx, torrentURL)
			if err != nil {
				if !errors.Is(err, errTorrentMetadataCoolingDown) {
					logging.Infof("taskruntime: inspect torrent candidate %q failed: %v", candidate.result.Title, err)
				}
				return
			}
			outMu.Lock()
			out[candidate.index] = inspectedCandidate{
				inspection: inspection,
				ok:         true,
			}
			outMu.Unlock()
		}(candidate, torrentURL)
	}
	wg.Wait()
	return out, inspectableCount
}

func inspectableTorrentURL(result indexer.SelectionCandidate) string {
	if result.ResourceID != "" {
		return preferredTorrentURL(result)
	}
	link := strings.TrimSpace(result.Link)
	if link == "" || strings.HasPrefix(strings.ToLower(link), "magnet:") {
		return ""
	}
	return link
}

func indexerPreferenceRank(result indexer.SelectionCandidate, rule config.CandidateSelectionRule) int {
	trackerID := strings.TrimSpace(result.TrackerID)
	if trackerID == "" {
		trackerID = strings.TrimSpace(result.Tracker)
	}
	for index, preferred := range rule.IndexerPreference.TrackerIDs {
		if strings.EqualFold(preferred, trackerID) || strings.EqualFold(preferred, result.Tracker) {
			return index
		}
	}
	return len(rule.IndexerPreference.TrackerIDs) + 1
}

// Positive ranks demote; negative ranks prefer. Exclusions and locks are
// applied independently so preferences cannot override them.
func titleMatchRank(title string, rule compiledRule) int {
	score := 0
	for i, clause := range rule.rule.TitleMatch.Clauses {
		if !matchesTitleClause(strings.ToLower(title), clause, rule.regexMatchers, i) {
			continue
		}
		switch clause.Effect {
		case config.TitleMatchEffectPrefer:
			score--
		case config.TitleMatchEffectDemote, config.TitleMatchEffectAvoid:
			score++
		}
	}
	return score
}
func singleVideoRank(candidate inspectedCandidate) int {
	if candidate.ok && candidate.inspection.SingleVideo {
		return 0
	}
	return 1
}
func torrentFileNameMatchRank(candidate inspectedCandidate, rule compiledRule) int {
	if !candidate.ok {
		return 0
	}
	score := 0
	for i, clause := range rule.rule.TorrentFileNameMatch.Clauses {
		if !matchesTorrentFileClause(candidate.inspection.Paths, clause, rule.regexMatchers, i) {
			continue
		}
		switch clause.Effect {
		case config.TorrentFileMatchEffectPrefer:
			score--
		case config.TorrentFileMatchEffectDemote, config.TorrentFileMatchEffectAvoid:
			score++
		}
	}
	return score
}
func filterCandidates(in []rankedCandidate, accept func(rankedCandidate) bool) []rankedCandidate {
	out := in[:0]
	for _, candidate := range in {
		if accept(candidate) {
			out = append(out, candidate)
		}
	}
	return out
}
func titleExclusionsAccept(title string, rules []compiledRule) bool {
	for _, rule := range rules {
		if rule.rule.Type != config.CandidateSelectionRuleTypeTitleMatch {
			continue
		}
		for i, clause := range rule.rule.TitleMatch.Clauses {
			matched := matchesTitleClause(strings.ToLower(title), clause, rule.regexMatchers, i)
			if clause.Effect == config.TitleMatchEffectExclude && matched {
				return false
			}
		}
	}
	return true
}
func hasFileExclusions(rules []compiledRule) bool {
	for _, rule := range rules {
		for _, clause := range rule.rule.TorrentFileNameMatch.Clauses {
			if clause.Effect == config.TorrentFileMatchEffectExclude {
				return true
			}
		}
	}
	return false
}
func fileExclusionsAccept(candidate inspectedCandidate, rules []compiledRule) bool {
	if !candidate.ok {
		return false
	}
	for _, rule := range rules {
		for i, clause := range rule.rule.TorrentFileNameMatch.Clauses {
			matched := matchesTorrentFileClause(candidate.inspection.Paths, clause, rule.regexMatchers, i)
			if clause.Effect == config.TorrentFileMatchEffectExclude && matched {
				return false
			}
		}
	}
	return true
}

// Locks match any clause, with no restriction when no candidate matches.
func titleLockMatches(title string, rules []compiledRule) bool {
	for _, rule := range rules {
		if rule.rule.Type != config.CandidateSelectionRuleTypeTitleMatch {
			continue
		}
		for i, clause := range rule.rule.TitleMatch.Clauses {
			if clause.Effect == config.TitleMatchEffectLock && matchesTitleClause(strings.ToLower(title), clause, rule.regexMatchers, i) {
				return true
			}
		}
	}
	return false
}

func fileLockMatches(candidate inspectedCandidate, rules []compiledRule) bool {
	if !candidate.ok {
		return false
	}
	for _, rule := range rules {
		for i, clause := range rule.rule.TorrentFileNameMatch.Clauses {
			if clause.Effect == config.TorrentFileMatchEffectLock && matchesTorrentFileClause(candidate.inspection.Paths, clause, rule.regexMatchers, i) {
				return true
			}
		}
	}
	return false
}

func matchesTitleClause(title string, clause config.TitleMatchClause, regexMatchers []*regexp.Regexp, index int) bool {
	switch clause.PatternMode {
	case config.TitleMatchPatternModeRegex:
		if index >= len(regexMatchers) || regexMatchers[index] == nil {
			return false
		}
		return regexMatchers[index].MatchString(title)
	default:
		return strings.Contains(title, strings.ToLower(clause.Pattern))
	}
}

func matchesTorrentFileClause(paths []string, clause config.TorrentFileNameMatchClause, regexMatchers []*regexp.Regexp, index int) bool {
	if len(paths) == 0 {
		return false
	}
	for _, item := range paths {
		normalized := strings.ToLower(item)
		switch clause.PatternMode {
		case config.TitleMatchPatternModeRegex:
			if index >= len(regexMatchers) || regexMatchers[index] == nil {
				continue
			}
			if regexMatchers[index].MatchString(normalized) {
				return true
			}
		default:
			if strings.Contains(normalized, strings.ToLower(clause.Pattern)) {
				return true
			}
		}
	}
	return false
}

func parsePublishDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC1123Z, value); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func titleSimilarityScore(query string, title string) int {
	if mediafiles.SameCode(query, title) {
		return 10000
	}
	normQuery := normalizeSimilarityText(query)
	normTitle := normalizeSimilarityText(title)
	if normQuery == "" || normTitle == "" {
		return 0
	}
	if normTitle == normQuery {
		return 10000
	}
	if strings.HasPrefix(normTitle, normQuery) {
		return 9000
	}
	if strings.Contains(normTitle, normQuery) {
		return 8000
	}

	queryTokens := tokenizeSimilarity(normQuery)
	titleTokens := tokenizeSimilarity(normTitle)
	shared := sharedTokenCount(queryTokens, titleTokens)
	score := shared * 100
	score += commonPrefixLength(normQuery, normTitle) * 10
	score += lcsLength(normQuery, normTitle)
	return score
}

func normalizeSimilarityText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r):
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
		default:
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func tokenizeSimilarity(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}

func sharedTokenCount(left []string, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	seen := make(map[string]int, len(right))
	for _, token := range right {
		seen[token]++
	}
	count := 0
	for _, token := range left {
		if seen[token] > 0 {
			count++
			seen[token]--
		}
	}
	return count
}

func commonPrefixLength(left string, right string) int {
	max := len(left)
	if len(right) < max {
		max = len(right)
	}
	count := 0
	for i := 0; i < max; i++ {
		if left[i] != right[i] {
			break
		}
		count++
	}
	return count
}

func lcsLength(left string, right string) int {
	if left == "" || right == "" {
		return 0
	}
	prev := make([]int, len(right)+1)
	curr := make([]int, len(right)+1)
	for i := 1; i <= len(left); i++ {
		for j := 1; j <= len(right); j++ {
			if left[i-1] == right[j-1] {
				curr[j] = prev[j-1] + 1
			} else if curr[j-1] > prev[j] {
				curr[j] = curr[j-1]
			} else {
				curr[j] = prev[j]
			}
		}
		copy(prev, curr)
		clear(curr)
	}
	return prev[len(right)]
}

func isVideoFilePath(value string) bool {
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(strings.TrimSpace(value))), ".")
	switch ext {
	case "mp4", "mkv", "avi", "wmv", "mov", "ts", "m2ts", "iso":
		return true
	default:
		return false
	}
}

func compareInts(left int, right int, direction config.CandidateSelectionDirection) int {
	switch direction {
	case config.CandidateSelectionDirectionAsc:
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	default:
		switch {
		case left > right:
			return -1
		case left < right:
			return 1
		default:
			return 0
		}
	}
}

func compareInt64s(left int64, right int64, direction config.CandidateSelectionDirection) int {
	switch direction {
	case config.CandidateSelectionDirectionAsc:
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	default:
		switch {
		case left > right:
			return -1
		case left < right:
			return 1
		default:
			return 0
		}
	}
}

func compareTimes(left time.Time, leftOK bool, right time.Time, rightOK bool, direction config.CandidateSelectionDirection) int {
	switch {
	case leftOK && !rightOK:
		return -1
	case !leftOK && rightOK:
		return 1
	case !leftOK && !rightOK:
		return 0
	}
	switch direction {
	case config.CandidateSelectionDirectionAsc:
		switch {
		case left.Before(right):
			return -1
		case left.After(right):
			return 1
		default:
			return 0
		}
	default:
		switch {
		case left.After(right):
			return -1
		case left.Before(right):
			return 1
		default:
			return 0
		}
	}
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
