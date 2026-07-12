import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const source = readFileSync(resolve("src/i18n/resources.ts"), "utf8");
const literal = source.replace(/^export const resources\s*=\s*/, "").replace(/\s+as const;\s*$/, "");
// resources.ts is a data-only object literal. Evaluating that literal keeps
// JSON generation dependency-free and deterministic.
const resources = Function(`"use strict"; return (${literal});`)();

for (const [locale, resource] of Object.entries(resources)) {
  const directory = resolve("public/locales", locale);
  const target = resolve(directory, "translation.json");
  const expected = `${JSON.stringify(resource.translation, null, 2)}\n`;
  if (process.argv.includes("--check")) {
    if (!existsSync(target) || readFileSync(target, "utf8") !== expected) {
      console.error(`Generated locale is stale: ${target}`);
      process.exitCode = 1;
    }
  } else {
    mkdirSync(directory, { recursive: true });
    writeFileSync(target, expected);
  }
}
