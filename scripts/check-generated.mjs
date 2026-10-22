import { readdir, readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";

async function snapshot(dir = "internal") {
  const result = {};
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const path = `${dir}/${entry.name}`;
    if (entry.isDirectory()) Object.assign(result, await snapshot(path));
    else if (path.includes("/sqlcgen/") || path.endsWith("_mock_test.go")) {
      result[path] = createHash("sha256").update(await readFile(path)).digest("hex");
    }
  }
  return result;
}
const before = await snapshot();
const generation = spawnSync("make", ["generate"], { stdio: "inherit" });
if (generation.status !== 0) process.exit(generation.status ?? 1);
const after = await snapshot();
const changed = [...new Set([...Object.keys(before), ...Object.keys(after)])].filter(path => before[path] !== after[path]);
if (changed.length) {
  console.error(`Generated files were stale. Review and commit the regenerated output:\n${changed.join("\n")}`);
  process.exit(1);
}
