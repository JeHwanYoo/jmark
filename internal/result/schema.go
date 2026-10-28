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
	return nil
}
