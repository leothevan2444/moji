/**
 * Performer-related utility functions.
 */

/**
 * Extract up to 2 leading characters as initials from a performer name.
 */
export function performerInitials(name: string) {
  return name
    .trim()
    .slice(0, 2)
    .toUpperCase();
}

/**
 * Resolve a performer image URL.
 * If the path is already absolute (http/https), return it as-is.
 * Otherwise, resolve it against the Stash base URL.
 */
export function performerImageURL(imagePath?: string | null, stashURL?: string | null) {
  if (!imagePath) return null;
  try {
    if (imagePath.startsWith("/api/images/")) return imagePath;
    if (/^https?:\/\//i.test(imagePath)) {
      return imagePath;
    }
    if (!stashURL) return imagePath;
    return new URL(imagePath, stashURL.endsWith("/") ? stashURL : `${stashURL}/`).toString();
  } catch {
    return imagePath;
  }
}

/** Build the matching performer detail URL in the configured Stash instance. */
export function stashPerformerURL(performerID: string, stashURL?: string | null) {
  if (!stashURL) return null;
  try {
    const url = new URL(stashURL);
    const basePath = url.pathname.replace(/\/$/, "");
    url.pathname = `${basePath}/performers/${encodeURIComponent(performerID)}`;
    url.search = "";
    url.hash = "";
    return url.toString();
  } catch {
    return null;
  }
}
