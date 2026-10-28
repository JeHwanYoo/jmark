// Package schema embeds the persisted Result contract.
package schema

import _ "embed"

//go:embed result.schema.json
var Result string
