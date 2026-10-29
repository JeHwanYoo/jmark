package report

import (
	"encoding/json"
	"strings"
)

type IndexRun struct {
	Project    string   `json:"project"`
	Run        string   `json:"run"`
	Runtimes   []string `json:"runtimes"`
	CapturedAt string   `json:"captured_at"`
	Href       string   `json:"href"`
}

func RenderIndex(runs []IndexRun, url string) (string, error) {
	if runs == nil {
		runs = []IndexRun{}
	}
	data, err := json.Marshal(runs)
	if err != nil {
		return "", err
	}
	projects := []string{}
	seen := map[string]bool{}
	for _, run := range runs {
		if !seen[run.Project] {
			seen[run.Project] = true
			projects = append(projects, run.Project)
		}
	}
	description := "Benchmark results measured with jmark under explicit, comparable conditions."
	if len(projects) > 0 {
		description = "Benchmark results measured with jmark: " + strings.Join(projects, ", ") + "."
	}
	return fillPageData("Results", description, string(data), "jmark-index-data", Options{URL: url}), nil
}
