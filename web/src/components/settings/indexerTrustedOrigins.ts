// This validates syntax only. The backend still resolves and checks every
// destination; trusting an origin grants credential use, never private access.
export function parseIndexerTrustedOrigins(text: string): { origins: string[]; error?: "invalidOrigin" | "tooManyOrigins" } {
  const origins = text.split(/\r?\n/).map(value => value.trim()).filter(Boolean);
  if (origins.length > 16) return { origins, error: "tooManyOrigins" };
  for (const value of origins) {
    try {
      const url = new URL(value);
      if (value.length > 4096 || !/^https?:\/\/[^/?#\\\s]+\/?$/.test(value) || !url.hostname || url.hostname.includes("*") || url.username || url.password || url.search || url.hash || url.pathname !== "/") return { origins, error: "invalidOrigin" };
    } catch { return { origins, error: "invalidOrigin" }; }
  }
  return { origins };
}
