package auth

import (
	"encoding/json"
	"io"
)

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
