# Activity levels and values

The **Activity level** section on a performer page shows five flame icons and a separate activity value. **Checks** shows the estimated next check, last check, and check status. Use **About activity levels and values** to open this page. Closing help returns you to the same performer.

## Value versus level

The **activity value** accumulates recent release contributions. It uses two decimal places and has no limit of 5 or 10. The **activity level** is a tier from 1 to 5 derived from that value. All five flame icons remain visible; the tier determines how many are lit.

Performer cards show only level icons at the lower left. An unknown rating is displayed as level 0: all five icons are unlit and crossed out by a horizontal line. Hover for the reason. Level 0 means a reliable rating is unavailable, not inactivity. A confirmed value of zero is still a normal level 1.

The rating measures recent release activity. Career length, retirement, popularity, library ownership, download policies, and task success do not directly change it. Equal release histories use equal rules.

## Calculating the value

Moji uses works linked to the performer at the preferred StashBox, including cached source records. This is not a local folder or the list of scenes already in Stash. Only valid **release dates** in the past 365 days count, never discovery or catalog entry dates. Future releases do not count. The configured application timezone determines today.

A valid release code identifies one work; otherwise the source scene ID does. If a code has multiple release dates, the latest valid date that has arrived is used once. Multiple releases on one date contribute only the highest daily weight:

- An ordinary release contributes 1.
- Only explicitly marked compilations or reissues contribute 0.2.
- Insufficient type information contributes 0.5.

A nonempty title without compilation or reissue markers counts as ordinary. Explicit title and tag markers identify compilations and reissues; a large cast alone does not. Inaccurate source metadata can affect the rating.

Each contribution halves every 60 days: an ordinary release today contributes 1, one 60 days ago contributes 0.5, and one 120 days ago contributes 0.25. Add the decayed daily contributions. Contributions older than 365 days are excluded.

```
Activity value = sum of [daily weight × 2^(-days since release / 60)]
```

## Levels and automatic intervals

- **Level 1**: value below 0.25; check every 30 days.
- **Level 2**: value from 0.25 to below 1; check every 14 days.
- **Level 3**: value from 1 to below 2.5; check every 7 days.
- **Level 4**: value from 2.5 to below 6; check every 3 days.
- **Level 5**: value of 6 or more; check every 24 hours.

A performer releasing their first ordinary work today usually starts at level 3. Continued releases accumulate contributions, while inactivity naturally reduces them. Compilations after retirement still make smaller contributions. Ratings are recalculated during successful source checks, rather than continuously reduced on the page.

Checks run automatically with Moji. The level determines the interval; it cannot be set manually. Manual checks also update the rating and schedule. Saved deadlines survive restarts. They are estimates: queues, downtime, or source failures can delay execution.

## Why a value has ≥

**≥ means a confirmed minimum, rather than an exact total.** For example, `≥ 15.15` means the works already read contribute at least approximately 15.15. Unverified records could increase the value.

Once confirmed contributions reach 6, the highest level and its 24-hour interval are certain. Moji can stop additional reads for scoring to reduce StashBox requests. The ≥ remains so a minimum is not presented as an exact total. Tier thresholds use the unrounded value; the displayed value uses two decimal places.

Confirming levels 1–4 requires a complete catalog or date-ordered coverage reaching the one-year boundary. Invalid dates or unverified performer links can also prevent a reliable rating. Each check reads at most six pages of 40 records. Additional scoring reads do not expand the automatic download scope into older works.

## Unavailable ratings and failures

Five unlit icons mean no reliable rating is available, not zero activity. The value field shows a specific state such as **Not assessed**, **Partial catalog**, or **Incomplete dates**. Hover for details. A successful check with insufficient evidence temporarily uses 48 hours.

Source failures preserve the previous rating. Consecutive failures retry after 1, 2, 4, 8, 16, then 24 hours, with further retries at most daily. A successful source read restores the normal interval. Later task creation failures do not invalidate a successful rating. Retry a failed page read using the icon beside the value; review check failures in **Checks**.
