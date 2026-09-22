package main

import (
	"encoding/json"
	"io"
)

func writeJSON(w io.Writer, r report) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(r)
}
