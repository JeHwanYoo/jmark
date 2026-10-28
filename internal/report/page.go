package report

import (
	_ "embed"
	"encoding/json"
	"html"
	"strings"
)

//go:embed page.html
var pageHTML string

//go:embed ui/dist/app.js
var appJS string

//go:embed ui/dist/app.css
var appCSS string

func fillPageData(project, data, dataID string, options Options) string {
	sources, _ := json.Marshal(options.Sources)
	return strings.NewReplacer(
		"@@PROJECT@@", html.EscapeString(project),
		"@@DATA_ID@@", dataID,
		"@@INDEX_HREF@@", html.EscapeString(options.IndexHref),
		"@@SOURCES@@", embedJSON(string(sources)),
		"@@DATA@@", embedJSON(data),
		"@@JS@@", strings.ReplaceAll(appJS, "</script", `<\/script`),
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
