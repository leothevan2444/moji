const ACCENTS: Record<string, string> = {
  a: "à", e: "ë", i: "ï", o: "ô", u: "ü", A: "Â", E: "Ë", I: "Ï", O: "Ö", U: "Û",
  c: "ç", C: "Ç", n: "ñ", N: "Ñ"
};

export function pseudoLocalize(message: string) {
  const parts = message.split(/(\{\{[^}]+\}\}|<[^>]+>)/g);
  const transformed = parts.map((part) => {
    if (part.startsWith("{{") || part.startsWith("<")) return part;
    return part.replace(/[aeiouAEIOUcnCN]/g, (character) => ACCENTS[character] ?? character).replace(/\b([A-Za-z]{4,})\b/g, "$1~");
  }).join("");
  return `［${transformed}］`;
}

export function pseudoResource<T>(value: T): T {
  if (typeof value === "string") return pseudoLocalize(value) as T;
  if (Array.isArray(value)) return value.map(pseudoResource) as T;
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, pseudoResource(item)])) as T;
  }
  return value;
}
