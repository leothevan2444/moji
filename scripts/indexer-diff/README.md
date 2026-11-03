# Actual interpreter comparison

This tool executes the pinned **unmodified Jackett Cardigann implementation**
against Moji using the same synthetic YAML and scripted HTTP responses. It
records request method/URL/body/headers, parsed common release fields, challenge
type/prompt/image SHA-256, and final torrent SHA-256 or magnet. It loads no production profiles, calls no websites,
and creates no Moji tasks or downloader operations.

Current cases cover public HTML/JSON, POST login/cookie verification, intermediate
downloads, POST download.before, testlinktorrent, and movie/music/book/TV query
fields. TV cases distinguish raw Ep from derived Episode, season packs, daily
dates, numeric normalization and invalid-query rejection. Text and image captcha
cases retain landing cookies and hidden fields, answer on the same interpreter
instance, then verify login, search and fetch with no duplicate landing/submission.
Pagination cases distinguish unset values from explicit zero and positive limits.
Numeric ID/year cases cover canonical formatting, missing fields, explicit empty
values and overflow. Declared array-style input names are checked on GET/POST.
Magnet cases cover download/magnet result fields, HTTP redirects, HTML selectors
and before-response selectors, preserving the entire URI including trackers.
The CJK keyword-filter case also checks .NET Unicode block semantics used by the
fixed community corpus. simpleCaptcha checks the automatic hash request and
ordered form submission, including submitme=X. Ten andmatch cases cover character
limits, filtered and fallback keywords, stopwords, missing terms, supported ID
search, the TV IMDb capability flag and UTF-16 boundaries. JSON count cases check
zero before absent rows, negative, positive, malformed and missing counts;
the HTML control confirms that this JSON-only property does not reject valid
HTML rows. Row grouping and date-header cases cover complete after groups,
sibling/parent scope, missing optional/required headers, invalid nearest headers
and preventing arbitrary ancestor traversal. Derived-query cases exercise all
15 host flags, empty string ID/genre presence, year zero, raw episode without a
season, and IMDb short-ID prefix behavior. Checkbox cases check raw output,
comparison with .True, and explicit true/false overrides. Select cases preserve
an explicit rule default and user override independently of option ordering,
and keep absent defaults or explicit empty selections empty.
Hash/title cases cover original and before pages, generated public/semi-private
result magnets and absence of an automatic public fallback for private results.
Case-insensitive parameter names are checked against the real host query model.
Date-filter cases compare explicit positive/negative offsets, seven fractional
digits, an invalid internal-whitespace control, and the canonical .Result.date
text. They check both intermediate strings and final publication timestamps.
JSON date cases compare ISO/Microsoft date scalar projections across 23 valid
and invalid inputs, arrays, case selection, jsonjoinarray and a filter/date
field chain. Literal selector text is not implicitly date-converted.
String-filter cases check first-character multi-character arguments, valid
supplementary-plane split/trim results, empty trim failure, and exact HTML
entities (quotation marks, Latin-1, BMP and supplementary text).
Query-string cases cover case-insensitive first-value lookup, semicolons,
invalid percent/UTF-8 escapes, and HTML attribute entity decoding. Further
cases check templated replacement, validate token order/delimiters, invariant
Unicode case and the Linux validfilename character set.
The 110 executed cases
include four explicitly expected rejections (invalid TV episode, captcha
answer rejected by the site, empty numeric ID/year and overflowing ID); they do not certify the community
library's live availability. Additional failure/cancellation
scenarios and other challenge implementations remain.

Relative dates are checked separately in `helper-cases/`: two fixed-clock
groups contain 38 inputs. Moji runs its actual timeago/reltime filters; the
pinned reference directly calls the unmodified DateTimeUtil.FromTimeAgo with
the same explicit relativeFrom clock. Reports are marked relative-date-helper,
including exact output text and declared invalid outcomes. They check elapsed
month/year intervals, float32 scaling, Unicode whitespace and range failures;
they are not additional complete search/login/download scenarios. No wall-clock
normalization or timestamp tolerance is applied to these helper results.

The user confirmed inheriting Jackett's public tracker defaults for generated
magnets. Both previously pending infohash/title cases are back in `cases/`, with
no difference exceptions. Site-provided magnets retain their original content.

andmatch follows Jackett's declarative row filter: apply its character-limit
argument, ignore the common words and/the/an, and require every remaining term
in the title unless a supported identifier search bypasses the check. Modern
Prowlarr instead applies host-level cleanup that ignores this argument and can
accept two terms in title or description. This deliberate choice preserves the
declared rule semantics and the Jackett comparison target; it is not a claim of
identical behavior across both hosts. Full Unicode case behavior and repeated
andmatch filters with different limits are not yet certified.

Checkbox values also follow the executed Jackett target: enabled is the text
True, matching the .True variable. The pinned Prowlarr variable-population code
uses the literal .True instead. Moji's .Today.Year keeps Prowlarr's current-year
behavior; Jackett uses the preceding year during January. Those source-level
host differences are explicit and do not imply identical interpreter behavior.
The January clock boundary has not been exercised against the actual reference.

The download cases explicitly record existing Moji behavior: selected downloads
use their originating details page as Referer, and testlinktorrent reuses the
validated bytes instead of downloading twice. Exact request differences have
named expectations; result/artifact comparisons still apply, and changed traces
invalidate those expectations. Reports and console summaries distinguish
explained differences from zero-difference cases.

Two query cases adopt modern Prowlarr behavior and declare exact differences
from Jackett: unset Limit/Offset remain unset, and POST input names retain their
declared brackets. The bounded production search service still passes its explicit
normalized limit and offset to Moji's interpreter. Positive offset rendering is
covered offline against Prowlarr's query model; Jackett's outer BaseIndexer skips
such requests for Cardigann rather than executing the interpreter.

## Prepare a reference

Install the fixed SDK development image explicitly:

```sh
docker pull mcr.microsoft.com/dotnet/sdk@sha256:6125486580cdf529a2a337d51c5557af08eaa04ef08b04f5b1d5a94f6bec7f1a
python3 scripts/indexer-diff/build_reference.py --download --work-dir /tmp/moji-indexer-reference
```

Alternatively use `--archive /path/to/archive.tar.gz` instead of `--download`.
The archive commit/digest, SDK image and all three NuGet lock-file digests are
pinned in `upstream.lock.json`. The executable and its two source-project
references have separate `packages.lock.json` files under `nuget-locks/`, covering
36 packages with resolved versions and content hashes. These versions match the
previously accepted reference graph. Builds copy the reviewed locks into the
temporary tree and use `dotnet restore --locked-mode`; they do not generate or
update the locks. The compile step runs in a separate network-disabled container
with `--no-restore`.
Use a new empty work directory; source extraction checks entry types, paths and
byte budgets. The explicit build restores public NuGet dependencies over the
network in a container with a read-only root and only the scratch mount. The
generated provenance records all output DLL digests and runner source digests;
comparisons reject changed binaries, pins, runner sources or dependency locks.
Changing a source or installed lock without updating the reviewed pins fails.
An intentional upstream/dependency update requires regenerating the locks in an
isolated source tree with the pinned SDK, reviewing the resolved package graph
and content hashes, then updating the lock snapshots/digests and rebuilding a
new reference directory. Never pass an unlocked fallback during normal builds.
Moji production has no Jackett or .NET runtime requirement from this tool.

## Run comparisons

Each comparison owns a temporary replay directory beneath the reference root.
Concurrent comparisons may share the immutable reference binaries; use a
different `--report` output for each invocation. Temporary inputs/results are
removed on completion or failure.

```sh
GOCACHE=/tmp/moji-go-cache python3 scripts/indexer-diff/compare.py \
  --reference-root /tmp/moji-indexer-reference --report /tmp/moji-indexer-diff.json
python3 scripts/indexer-diff/compare_test.py
python3 scripts/indexer-diff/build_reference_test.py
```

Run the two fixed-clock helper groups separately, as CI does:

```sh
GOCACHE=/tmp/moji-go-cache python3 scripts/indexer-diff/compare.py \
  --reference-root /tmp/moji-indexer-reference \
  --case scripts/indexer-diff/helper-cases/relative-dates/scenario.json \
  --case scripts/indexer-diff/helper-cases/relative-dates-zero-ticks/scenario.json \
  --report /tmp/moji-relative-dates.json
```

Use repeated `--case path/to/scenario.json` to select cases. Moji's replay command
has only an in-memory fixture RoundTripper; unscripted paths fail without a
network fallback. Jackett runs in a network-disabled container, uses its real
Cardigann class and WebClient preparation, with only `Run` replaced by response
dispatch. POST bytes are produced by the upstream protected serialization
helper. The harness therefore tests interpreter semantics and that HTTP client
boundary; it does not certify live TLS, proxies, rate limiting or browser solving.
Containers have unique owned names and are removed on completion or timeout.

Jackett's download API returns UTF-8 bytes for intermediate/redirected magnets.
The runner exposes a `magnet:` byte result in the magnet branch of the report;
other bytes retain the torrent SHA-256 branch. It does not reorder the URI,
remove tracker parameters, or compare only the content hash.

For result MagnetUri values, the runner uses the pinned Newtonsoft.Json typed
Uri serialization used by Jackett's API, rather than Uri.ToString's display
form. An isolated SDK probe confirmed that the display form unescapes Unicode
while the API retains its original percent encoding. Direct-result artifacts
use the same API projection. Download-returned magnet bytes stay untouched, and
the Python checker still treats encoded and unencoded URI strings as different.

Case files reference local `definition.yml` and response files within their case
directory, or literal synthetic body text. The orchestrator resolves them into
one bounded JSON input with base64 bodies for both engines. Settings use the
interpreter's string-to-string boundary, including "true"/"false"; the checker
rejects boolean/number JSON values before launching either interpreter. Keep all cases
synthetic. Reports include credentials/cookies from those fixtures; use temporary
private reports and never point the tool at user configuration or private captures.

Manual challenge cases set `authenticate: true` and a bounded nonempty
`challengeAnswer`. The Jackett runner invokes its actual manual setup API and
supplies the answer through its dynamic captcha configuration before applying
configuration; Moji resumes the pending form through `AnswerChallenge`.
Image presentation compares bytes by digest; Jackett's public configuration
does not retain the response MIME type, so that property is not certified by
this comparison. These cases exercise interpreter continuation, not browser UI
or recognition of the synthetic image bytes.

UTC is fixed for the reference environment; shipped cases avoid relative time
expressions. Moji's clock is scenario-controlled, but Jackett still uses its real
clock. Relative date comparisons need an additional clock strategy.

## Difference policy

Raw engine output is retained. Comparison only normalizes header name casing
and equivalent UTC fractional formatting. Exact implicit `Accept: */*` and
`Accept-Language: *` headers are listed separately as transport defaults. Custom
values remain differences. No URL or body byte differences are silently ignored.

An intentional difference can be declared in a case's `expectedDifferences` as
an exact JSON-pointer path, both values, and a reason. Changed values or stale
expectations fail the case. Engine errors fail by default, even when both engines
fail similarly. An error scenario must declare exact nonempty errors for both
engines in `expectedErrors`; a changed error or unexpected success fails. After
matching errors, absent result lists are treated as empty; request sequences,
results and artifacts still undergo comparison. Unexpected differences produce
exit code 1 and a private JSON report. Empty or missing scenarios do not pass.

The main CI runs Go replay tests and checker tests offline. Its reusable
`indexer-compatibility.yml` job builds the pinned reference, runs the actual
comparison and downloads/compiles the fixed v11 corpus. Publication waits for
that job along with Web, Go and Docker checks. The compatibility workflow can
also run manually. Report artifacts distinguish static and executed coverage.
Workflow source configuration alone is not evidence that remote Actions ran.
