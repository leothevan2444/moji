# Resource selection

Configure **Settings → Tasks & subscriptions → Resource selection**. The rule library describes matching conditions. Rule groups reuse those rules and decide which resources are eligible and which tier comes first. There are no built-in version, subtitle or resolution preferences, keyword dictionaries, or media-label classifiers.

Without a group, Moji applies no additional user filtering. Complete target-film identity, valid torrent/magnet and duplicate-task checks always remain active. Adding a group requires selecting the active group; automatic sourcing uses that single active group. Search previews can temporarily select another group without changing the saved active group.

The settings page has two cards: the rule library and rule groups. Click anywhere on a rule or group card to open its editor, except its action controls. The group card combines its use status and selection button at the bottom left; the checked circle marks the active group. Active groups use the same neutral card border as other groups; hover and keyboard focus retain their normal feedback. There is no separate active-group dropdown. Rule and group editors display their names as editable window titles; rename directly in the title instead of a separate name field. Opening an editor focuses the dialog container, not the title. Tab still reaches the title for keyboard editing. Window title fields use about 40% of the available header width. Changes use the existing automatic save flow; closing an editor keeps edits, and failures stay visible with retry available.

Rule cards include a drag handle and deletion action. Drag with a mouse or touch, or use arrow keys on the handle to organize the library. Group cards also have a borderless drag handle at the top left, beside the group name; rearranging them preserves the active group. Order is saved but does not change filtering or resource priority. Long names remain available through the name button's tooltip and editor. Groups show admission, exclusion and tier counts together, then an ordered preview of up to three priority names. The footer combines the use status, fallback state and deletion action. Group cards share a fixed height, reserving space for three preview names and a fourth “more” row even when fewer tiers exist. Long group names use at most two lines, with the full name available through the tooltip. Activation and deletion buttons are separate from the edit button. Icon-only actions are square; group handles and the bottom-left status control align with the content inset. Keyboard focus on the edit action outlines the whole card.

Deleting a rule or group always opens a confirmation dialog, from either a card or its editor. For a referenced rule, the confirmation lists affected groups and explains that all references will be removed. Priority tiers remain, including tiers left empty. Deleting a group keeps its rule definitions. If the active group is deleted, the confirmation explains which remaining group will become active; deleting the last group removes additional rule filtering.

## Rule library

A rule has a stable ID, a name, and matching conditions. **Any condition matching makes the rule match.** This is fixed behavior; users cannot choose all/any within a rule. There is no condition-count limit, subject to the overall policy size budget. Each condition selects its own field and supplies a text expression or numeric range. There are no condition operators or individual negation options.

For example, a “Subtitle label” rule can list several title expressions. Any matching expression is sufficient. If no condition matches and at least one is unknown, the rule remains unknown. The rule is false only when every condition is known and false. Tier references always require a positive match; unknown never counts as a match.

A rule has no priority, filtering action or enable switch. A rule affects a decision only when the selected group references it. The same rule can appear in several groups; editing it affects every referencing group. The editor footer and deletion confirmation show those groups. Each condition occupies one compact row: choose its field, enter a text expression or numeric bounds, and use the trash button to remove it. The plus icon at the top right of the condition heading adds another condition. The last condition cannot be deleted. Expression and range help is available from the condition heading. Blank text expressions and numeric ranges with neither bound are ignored when saving; unfinished rows remain editable in the current draft. A saved rule may have no conditions and then never matches. Invalid nonempty expressions and ranges still require correction.

| Match field | Source | Condition value |
| --- | --- | --- |
| Search title | Title returned by the indexer | Literal text or `/regular expression/` |
| Total torrent size | Size returned by the indexer | Minimum and/or maximum |
| Seeders | Count returned by the indexer | Minimum and/or maximum |

Text matching is case-insensitive. Ordinary text is a literal substring: `a.b` matches a literal dot. Wrap a regular expression in `/…/`, such as `/alpha|beta/` or `/^ABCD-123$/`. No manual type selector is needed. Regular expressions use Go RE2 syntax; invalid expressions fail validation rather than falling back to literal text.

Numeric bounds are inclusive. Set either or both bounds; equal bounds match an exact value. Bounds are nonnegative integers in YAML/API. The size editor accepts decimal GiB and saves integer bytes. Missing indexer sizes or seeder counts remain unknown; neither zero nor another inferred value is substituted, and matching never fetches metadata to fill them.

Write your own label expressions for version, subtitle or resolution selection. A name such as “No modification label” describes a naming condition, not a guarantee about the actual picture. Resource rules do not inspect torrent names, file paths, video streams or subtitle files. Download submission and subsequent file handling retain their existing protections.

## Rule groups

A group has a pre-filter, ordered priority tiers, and an **Allow fallback selection** switch. The switch stays beside its label in the editor’s bottom-left footer, with its explanation available through the info tooltip. Combination behavior is fixed; there are no all/any selectors:

1. **Pre-filter → Admission rules**: matching any referenced rule satisfies admission. An empty list imposes no requirement.
2. **Pre-filter → Excluded**: matching any referenced rule rejects the resource. An empty list excludes nothing. Exclusion takes precedence over admission.
3. **Priority tiers**: every rule reference in a tier must match; the first matching tier determines priority. Empty tiers are valid to save and never match.

Admission and exclusion share a compact pre-filter area, side by side on desktop and stacked on small screens. Use the plus icon at the top right of each filter to open the native rule picker. Add rules from the picker or remove a tag with its X. These tags have no drag handles and always follow the current rule-library order, including live updates when the library is reordered. Filter titles and hints stay together with a 2px gap. These lists reference positive rule IDs directly. A rule cannot belong to both admission and exclusion in the same group. Exclusion and priority references are also mutually exclusive, with no inverse priority references. Pickers in both directions omit conflicting rules; remove a reference to make it available elsewhere. The server also rejects conflicting configurations.

Priority tiers are edited as cards inside the group editor. Cards use two columns on desktop and one on small screens. The add-tier action shares the priority explanation row rather than taking a separate row. Each card uses its editable name as the title, with a drag handle beside it and plus/delete icons at the top right. The plus icon opens the native rule picker. Rule tags sit directly below the card title without a repeated section heading; there is no separate tier editor window. New tiers start without rule references and may stay empty. Drag with a mouse or touch, or use arrow keys on the handle to change priority. Tier tags have an X to remove the reference. All references match positively; there are no match/exclude relations or action menus. Every referenced rule must match, and unknown information cannot satisfy a tier.

With **Allow fallback selection** enabled, resources that match no priority tier can still be selected after passing the pre-filter. They rank below every matching priority tier and never trigger first-tier early stopping. Disabling fallback restricts automatic selection to resources matching a tier. Fallback is a boolean on the group; it has no saved tier, tier ID, name or rule references, and never bypasses exclusions or unresolved necessary information.

A group may have zero priority tiers. With fallback enabled, it selects among resources passing the pre-filter; with fallback disabled, it selects no resources automatically. New groups start with fallback disabled. Existing configurations with an explicit fallback tier must be backed up and manually converted to the switch. Only one group applies per selection; groups are not stacked or nested.

Changing tier order changes priority. Rule-library order does not. Between resources in the same tier, positive seeder counts come first, followed by unknown counts and zero seeders; site order and stable result order settle remaining ties. Counts do not guarantee transfer speed.

## Search and missing information

Automatic sourcing searches sites in configured order and retains candidates across sites. A matching first-tier candidate permits early stopping based on indexer information. Otherwise, after searching, Moji selects the highest available eligible tier, or a fallback candidate if enabled. Equal-tier resources use positive seeder counts, unknown counts and zero seeders, then configured site order and stable result order. There is no file-manageability ranking.

A search round lasts at most ten minutes and retains at most 100 candidates per site and 1,000 overall. Search failures and limits produce incomplete coverage. There are no per-site metadata inspection quotas, acquisition budgets, file-check workers or metadata validation loops in resource selection. Insufficient indexer information cannot silently satisfy admission, exclusions or a higher priority tier.

The result means “matches the configured rules using indexer information”; it does not certify actual content, availability or download success. After selecting one resource, Moji retains the existing acquisition and submission validation for that single resource. Failure does not cause a loop over other candidates. A new download-preparation stage and ordered candidate acquisition/validation are deliberately deferred.

The library allows 64 rules and 16 groups, up to 16 real tiers per group and 32 references per tier. Global lists can each reference every library rule once. Conditions have no count limit, subject to the complete serialized policy budget of 192 KiB. IDs are stable and unique; names are limited to 160 bytes and expression bodies to 512 bytes.

## Preview and explanations

Choose a target film and optionally a group, then preview. Preview matching uses the existing search snapshot and returns synchronously. It does not acquire torrent metadata, create background jobs, poll or offer cancellation of file inspections. It evaluates the complete search result set before projecting the requested page. Group choice remains in the URL; preview results are temporary view state, so refresh requires clicking preview again. Task recommendations use the same immediate matching.

Explanations show the group, first matching tier or fallback, rule conditions, match results and indexer evidence. Missing information remains visible. Evidence is bounded, and omitted entries are counted. The core database retains the accepted resource identity and recovery material in its existing task records; it stores no separate selection summary or report reference. Reports, including candidate comparisons, policy snapshots and evidence, belong entirely to runtime diagnostics, limited to 256 KiB each. They follow the existing 14-day / latest 100,000 diagnostics retention policy. The task detail shows the explanation only while its report is available. Expiry, runtime repair or report-storage failure does not undo a selection or block task recovery. Available historical reports remain readable, including their old file evidence, without reinterpreting them under new rules. Editing rules affects later previews and selections, not accepted downloads.

## YAML example

This example uses arbitrary user-maintained labels. It is not an automatically installed preset.

```yaml
tasks:
  torrent_selection:
    version: 7
    active_group: daily
    rules:
      - id: unwanted
        name: Unwanted label
        conditions:
          - scope: TITLE
            value: unwanted-label
      - id: wanted
        name: Preferred label
        conditions:
          - scope: TITLE
            value: '/preferred-label|another-label/'
      - id: size
        name: Size between 1 and 20 GiB
        conditions:
          - scope: SIZE
            minimum: "1073741824"
            maximum: "21474836480"
    groups:
      - id: daily
        name: Daily collection
        fallback: true
        require: [size]
        exclude: [unwanted]
        tiers:
          - id: first
            name: Preferred names
            rules: [wanted]
```

To apply no additional filtering, use `active_group: ""`, `groups: []` and an empty or unused rule library. Resource policy version 7 permits only TITLE, SIZE and SEEDERS. It has no `per_site` or `max_fetches`; file scopes are rejected rather than ignored. Back up and manually remove unsupported file conditions. A rule that becomes empty can remain saved and never matches; empty tiers may remain saved and never match. Do not silently reinterpret file conditions as title conditions.

The active selector uses YAML `active_group` and GraphQL `activeGroup`. Global `require` and `exclude` contain rule IDs and always use any-match. Tier `rules` also contain scalar rule IDs and use positive all-match. Policy v7 rejects old reference objects and v6 settings rather than silently reinterpreting negation. Back up and manually convert v6 configurations; changing a formerly negated reference to a positive reference changes its meaning. Historical saved decisions remain readable and are not re-evaluated. Group `fallback` is a boolean and preserves explicit false on updates. Numeric condition bounds are exact integer strings. Configuration loading never performs automatic conversion.
