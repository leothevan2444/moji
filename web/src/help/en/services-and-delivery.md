# Service connections and Stash delivery

These settings define how Moji reaches its dependencies and how a completed download becomes visible to Stash. Changes save automatically after a 500 ms pause. Only changed fields are submitted; failed saves retain your edits and offer retry.

## Service connections

Manage built-in sites under **Indexers**, with optional Jackett. qBittorrent is under **Downloader** and Stash under **Stash**.

Configure the base URL and credentials for Stash, qBittorrent, and Jackett if you use it. Moji derives the required API endpoints from those base URLs. A service is **configured** when its minimum fields are present and **ready** only after a recent probe succeeds.

qBittorrent's default save path, category, and tags are applied to new tasks unless an operation supplies an override. The Jackett dashboard password is used when torrent files require authenticated retrieval. Credentials are returned to the current settings UI, so access to Moji should be restricted.

## Definition library and update policy

Built-in indexing uses the independently downloaded Prowlarr v11 definition library. For the first installation, choose **Download definitions** under **Indexers > Definitions**. Later checks and updates do not require updating Moji itself.

The default is to check every 24 hours and apply updates manually. **Update policy** supports a check interval of 1–168 hours and automatic application. When enabled, new versions are validated and applied at the next scheduled check. Turning it off does not cancel an update already in progress; use the job's cancel action to stop it.

Conflicts preserve your edits. **Load current settings** asks for confirmation before discarding unsaved changes and reading the latest server policy.


Home’s **Indexers** section shows the current failed or interrupted definition job. **View definitions** opens its status and actions. Reloading or reconnecting restores server state; a subsequent job replaces the old notice. Reading or retrying Home status does not check for updates or download definitions.

## Test a site

Choose **Test** in the site list or built-in site configuration. It checks connectivity immediately and displays Testing, Connected or Failed on the button. Hover for failure details. **Test all** includes disabled sites and checks at most two sites concurrently, retaining each result. Tests do not log in, search, acquire resources or create download tasks.

Built-in tests request the site address; Jackett tests check the corresponding Jackett API. A reachable Jackett API does not verify upstream site search or acquisition. The configuration window tests the current draft without saving it; editing parameters clears the result. Reloading requires a new test, and leaving settings cancels unfinished checks.

## Log in to a site

In **Indexers > Sites**, choose **Log in to site** for a built-in site. Login uses saved settings; save an editing draft first. Opening the dialog only reads an existing challenge. Select **Start login** to authenticate, then **Submit answer** if the site requests text or image verification. The answer is cleared on submission and is not saved in settings.

Reopen the dialog after closing or refreshing the page to recover a challenge that is still valid. Expiry or a Moji restart requires a fresh login. **Restart login** clears the old verification first. **Cancel login** can stop an in-flight answer and release the flow. The resulting session is used for searches and resource acquisition, which are still checked separately.

If the definition has no separate login flow, configure a cookie if the site requires one. Manage Jackett authentication in Jackett.

## Additional trusted origins

Open a built-in site editor and expand **Advanced settings** only when authentication or resource retrieval needs another origin. Enter one full HTTP(S) origin per line, such as `https://download.example` or `https://auth.example:8443`, up to 16. Paths, query parameters, fragments, credentials and wildcards are not accepted. Clear the field and save to revoke all additional origins for new operations. Already accepted tasks retain their frozen source settings.

These origins may receive site credentials and form submissions, and their cookies may update the site session. Cookie domain and secure rules still apply; listing an unrelated host does not copy account cookies to it. Unlisted origins do not receive session cookies and cannot update them, including subdomains covered by an account's domain cookie. Trusting an origin does not permit private-network access or HTTPS downgrades. Connectivity checks do not change the saved trust list or verify cross-origin authentication and retrieval.

## Browser verification and network profiles

Under **Indexers > Network & cache**, add a profile with a proxy, a FlareSolverr URL, or both, then select it in the site editor. Enter the service base URL (for example `http://flaresolverr:8191`) without `/v1`. URLs are not returned to the editor: leave blank to keep them, or select the explicit clear option and save. Saving does not test connectivity.

FlareSolverr is an optional trusted external browser service. For recognized login or search challenges, Moji sends the site's cookies and the selected proxy credentials, then verifies the original request with the returned cookies and browser User-Agent. Moji does not automatically replay torrent acquisition or its preceding pages; a remaining challenge needs attention. Moji checks the delegated destination and returned origin but cannot constrain every navigation inside the external browser.

**Browser verification timeout** defaults to 90 seconds and accepts 10–300 seconds. The remaining login, search or test deadline can shorten it; this setting does not extend the whole operation. Changes apply to subsequent new operations; tasks with frozen sources retain their original parameters.

## Ingest

Moji makes the task save directory relative to the **qB download root**, then appends it to the **Moji download root** to read selected files. It filters, deduplicates, and merges the movie into `code/code.mkv` under the **Moji library root**, together with retained subtitles.

The **Stash library root** must expose the same files as the Moji library root. Moji appends the movie code directory to this root and asks Stash to scan the delivered movie.

All four roots use the named service's filesystem perspective and must match container mounts. **Copy** retains downloaded sources. **Move** stops the qB task and deletes selected sources only after the entire movie and subtitles are published. Existing movies are not overwritten. The Moji runtime requires `ffmpeg` and `ffprobe`.
