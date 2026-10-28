package report

import "encoding/json"

type IndexRun struct {
	Project    string   `json:"project"`
	Run        string   `json:"run"`
	Runtimes   []string `json:"runtimes"`
	CapturedAt string   `json:"captured_at"`
	Href       string   `json:"href"`
}

func RenderIndex(runs []IndexRun) (string, error) {
	if runs == nil {
		runs = []IndexRun{}
	}
	data, err := json.Marshal(runs)
	if err != nil {
		return "", err
	}
	return fillPageData("Results", string(data), "jmark-index-data", Options{}), nil
}
