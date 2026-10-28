# Go host. Recipes invoke via go run so the tree stays binary-free.
jmark := "go run ./cmd/jmark"

# Host deps, registry, project layouts, and probe availability.
doctor:
    @{{jmark}} doctor

# Project ids from benches/*/AGENTS.md, one per line.
list:
    @{{jmark}} list

# List PATH availability; lock and install delegate to mise.
runtimes *args:
    @{{jmark}} runtimes {{args}}

# Scaffold benches/<id> for each named runtime's family. Idempotent, never overwrites.
create id +runtimes:
    @{{jmark}} create {{id}} {{runtimes}}

# stream → Result NDJSON on stdout (pipeline).
# ndjson → artifacts/private/<id>/<timestamp>/result.ndjson; path on stdout.
# html → artifacts/private/<id>/<timestamp>/{result.ndjson,report.html}; html path on stdout.
# Measure [<id>]; omit <id> on a terminal to choose project, runtime, and format. Optional --runtime, --format (default stream).
bench *args:
    @{{jmark}} bench {{args}}

# fzf multi-select artifacts/<private|public>/<id>/<timestamp> runs: Enter = chart compare, Ctrl-X = delete.
open *args:
    @{{jmark}} open {{args}}

# fzf multi-select artifacts/<private|public>/<id>/<timestamp> runs and delete after y confirm.
delete *args:
    @{{jmark}} delete {{args}}

# Rebuild the embedded report after editing Svelte components.
ui-build:
    cd internal/report/ui && npm ci && npm run check && npm test && npm run build

# Move selected private runs into Git-visible public storage.
public *args:
    @{{jmark}} public {{args}}

# Move selected public runs back to ignored private storage.
private *args:
    @{{jmark}} private {{args}}

# Build the public index and reports for GitHub Pages.
site:
    @{{jmark}} site
