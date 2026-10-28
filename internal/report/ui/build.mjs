import { build } from "esbuild";
import { compile } from "svelte/compiler";
import { readFile, readdir, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
await build({
  entryPoints: ["src/main.js"],
  bundle: true,
  loader: { ".svg": "dataurl" },
  minify: true,
  format: "iife",
  outfile: "dist/app.js",
  conditions: ["browser"],
  legalComments: "eof",
  plugins: [
    {
      name: "svelte",
      setup(build) {
        build.onLoad({ filter: /\.svelte$/ }, async ({ path }) => {
          const result = compile(await readFile(path, "utf8"), {
            filename: path,
            css: "injected",
            generate: "client",
          });
          for (const warning of result.warnings) console.warn(warning.message);
          return { contents: result.js.code, loader: "js" };
        });
      },
    },
  ],
});

async function sourceFiles(directory) {
  const files = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = `${directory}/${entry.name}`;
    if (entry.isDirectory()) files.push(...(await sourceFiles(path)));
    else files.push(path);
  }
  return files;
}

// Record both inputs and outputs so host tests detect stale or modified bundles.
const files = [
  "build.mjs",
  "package.json",
  "package-lock.json",
  ...(await sourceFiles("src")),
  "dist/app.js",
  "dist/app.css",
].sort();
const hashes = {};
for (const path of files) {
  hashes[path] = createHash("sha256")
    .update(await readFile(path))
    .digest("hex");
}
await writeFile("dist/manifest.json", `${JSON.stringify(hashes, null, 2)}\n`);
