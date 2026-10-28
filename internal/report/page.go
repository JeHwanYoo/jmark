package report

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"html"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

//go:embed page.html
var pageHTML string

//go:embed ui/dist/app.js
var appJS string

//go:embed ui/dist/app.css
var appCSS string

//go:embed ui/src/assets/favicon.ico
var favicon []byte

// metricCatalog lets the guide explain metrics a run did not request.
func metricCatalog() []byte {
	type entry struct {
		Label    string   `json:"label"`
		Readings []string `json:"readings"`
	}
	metrics := map[string]entry{}
	for id, d := range benchmark.Metrics {
		metrics[id] = entry{d.Label, d.Readings}
	}
	catalog, _ := json.Marshal(map[string]any{"metrics": metrics, "readings": benchmark.Readings})
	return catalog
}

func fillPageData(project, data, dataID string, options Options) string {
	sources, _ := json.Marshal(options.Sources)
	return strings.NewReplacer(
		"@@PROJECT@@", html.EscapeString(project),
		"@@DATA_ID@@", dataID,
		"@@INDEX_HREF@@", html.EscapeString(options.IndexHref),
		"@@SOURCES@@", embedJSON(string(sources)),
		"@@METRICS@@", embedJSON(string(metricCatalog())),
		"@@DATA@@", embedJSON(data),
		"@@JS@@", strings.ReplaceAll(appJS, "</script", `<\/script`),
		"@@FAVICON@@", base64.StdEncoding.EncodeToString(favicon),
		"@@CSS@@", appCSS,
	).Replace(pageHTML)
}

// Escape HTML-sensitive characters even inside application/json script elements.
func embedJSON(data string) string {
	var b strings.Builder
	for _, r := range data {
		switch r {
		case '<':
			b.WriteString(`\u003c`)
		case '>':
			b.WriteString(`\u003e`)
		case '&':
			b.WriteString(`\u0026`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
