package importer

import (
	"encoding/json"
	"strings"
)

// jsonDecoder wraps strings.NewReader for cleaner readability in the
// importer's hot path. Lives in its own file so the dependency on
// encoding/json doesn't leak into the package surface.
func jsonDecoder(line string) *json.Decoder {
	return json.NewDecoder(strings.NewReader(line))
}
