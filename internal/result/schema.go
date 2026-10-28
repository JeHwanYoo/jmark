package result

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/JeHwanYoo/jmark/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var compiledSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schema.Result))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	const location = "https://jmark.local/schema/result.schema.json"
	if err := compiler.AddResource(location, doc); err != nil {
		return nil, err
	}
	return compiler.Compile(location)
})

// CheckSchema validates the emitted JSON against the embedded Result schema.
func CheckSchema(v map[string]any) error {
	contract, err := compiledSchema()
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	// Normalize Go integers and RawMessage metadata while rejecting non-JSON numbers.
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := contract.Validate(doc); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	return validateReferences(v)
}

// validateReferences ensures the schema's open metric registry stays internally consistent.
func validateReferences(v map[string]any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var doc struct {
		Metrics map[string]json.RawMessage `json:"metrics"`
		Results []struct {
			Metrics map[string]json.RawMessage `json:"metrics"`
		} `json:"results"`
	}
	if err = json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	for _, row := range doc.Results {
		if len(row.Metrics) != len(doc.Metrics) {
			return fmt.Errorf("metric definitions and results differ")
		}
		for id := range row.Metrics {
			if _, ok := doc.Metrics[id]; !ok {
				return fmt.Errorf("undefined metric %s", id)
			}
		}
	}
	return nil
}
