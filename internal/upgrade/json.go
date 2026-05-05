package upgrade

import (
	"encoding/json"
	"io"
)

// jsonDecode reads r and decodes into v. Kept as a package-level helper
// so the rest of the package doesn't need to import encoding/json directly.
func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
