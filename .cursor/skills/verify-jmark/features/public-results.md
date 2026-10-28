# Public results

Saved runs begin in private storage. Explicit visibility changes select the runs included in Git and the generated public website.

## Sub-features

- Default private storage and Git ignore rules.
- Public/private moves preserving result and source bytes.
- Public-only static site output with an index and report navigation.
- Saved source Markdown, syntax highlighting, and download.

## How to get to it (user POV)

Save a benchmark with `--format html` or `--format ndjson`. Use `just public` to
select runs, or pass `<project>/<run>`. Run `just site` to generate the public site.

## Driving it with just

Capture commands with `helpers/capture.sh`.

1. Save a new benchmark. Require a path under `artifacts/private/` and `sources.md`
   containing fenced code blocks from the captured sources.
2. Move only that new run with `just public <project>/<run>`. Verify Git does not
   ignore its result file and that the file bytes are unchanged.
3. Run `just site`. Open `.jmark/site/index.html` through a local static server.
   Search for the run, open its report, and use `←` to return to the index.
4. Verify syntax highlighting and compare downloaded Markdown with `sources.md`.
5. Run `just private <project>/<run>` and rebuild the site. Require the run to be
   excluded from site output and its private files to be ignored by Git.

## Gotchas

- Never publish existing user runs as a verification shortcut.
- Public moves do not stage, commit, or push files.
- Previously committed results remain in Git history after moving to private.
- Historical runs without source snapshots must not receive current working code.
- GitHub Pages deployment needs a remote repository and its publishing source set
  to GitHub Actions; local site generation does not require either.
