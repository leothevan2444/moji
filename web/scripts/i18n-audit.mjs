import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const root = new URL("../src/", import.meta.url).pathname;
const ignored = new Set(["i18n/resources.ts"]);
const findings = [];

function visit(path) {
  for (const name of readdirSync(path)) {
    const target = join(path, name);
    if (statSync(target).isDirectory()) {
      if (name !== "generated" && name !== "help") visit(target);
      continue;
    }
    if (!/\.(ts|tsx)$/.test(name)) continue;
    const short = relative(root, target);
    if (ignored.has(short)) continue;
    readFileSync(target, "utf8").split("\n").forEach((line, index) => {
      if (/[\u3400-\u9fff]/u.test(line) && !line.trimStart().startsWith("//") && !line.trimStart().startsWith("*")) {
        findings.push(`${short}:${index + 1}: ${line.trim()}`);
      }
    });
  }
}

visit(root);
console.log(`i18n audit: ${findings.length} possible hard-coded CJK UI strings`);
for (const finding of findings) console.log(finding);
if (process.argv.includes("--strict") && findings.length) process.exitCode = 1;
