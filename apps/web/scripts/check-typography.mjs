import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const rules = [
  {
    files: [
      "src/components/ui/input.tsx",
      "src/components/ui/textarea.tsx",
      "src/components/ui/native-select.tsx",
      "src/components/list/list-search.tsx",
    ],
    pattern: /\btext-(?:base|lg)\b/,
    message: "kontrol form wajib memakai text-sm",
  },
  {
    files: [
      "src/components/dashboard-view.tsx",
      "src/components/dashboard-operations.tsx",
    ],
    pattern: /type-overline|text-lg\s+font-semibold/,
    message: "label KPI wajib text-sm dan nilai KPI wajib text-2xl",
  },
];

let violations = 0;

for (const rule of rules) {
  for (const file of rule.files) {
    const source = readFileSync(path.join(root, file), "utf8");
    source.split(/\r?\n/).forEach((line, index) => {
      if (rule.pattern.test(line)) {
        violations += 1;
        console.error(`${file}:${index + 1}: ${rule.message}\n  ${line.trim()}`);
      }
    });
  }
}

if (violations > 0) {
  console.error(`\n${violations} pelanggaran tipografi ditemukan.`);
  process.exit(1);
}

console.log("Typography check lulus.");
