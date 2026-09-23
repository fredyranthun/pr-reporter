package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func writeJSON(w io.Writer, r report) error {
	return writeJSONFields(w, r, nil)
}

// --fields projects PR objects only. The outer report retains completeness,
// counters, repository results, and diagnostics even for partial collection.
func writeJSONFields(w io.Writer, r report, fields []string) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	if fields == nil {
		return e.Encode(r)
	}
	prs := make([]selectedPR, 0, len(r.PullRequests))
	for _, p := range r.PullRequests {
		prs = append(prs, selectedPR{value: p, fields: fields})
	}
	return e.Encode(struct {
		report
		PullRequests []selectedPR `json:"pull_requests"`
	}{report: r, PullRequests: prs})
}

type selectedPR struct {
	value  pullRequest
	fields []string
}

func (p selectedPR) MarshalJSON() ([]byte, error) {
	full, err := json.Marshal(p.value)
	if err != nil {
		return nil, err
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(full, &values); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteByte('{')
	for i, name := range p.fields {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(name) // validated ASCII JSON keys from prFields
		b.Write(key)
		b.WriteByte(':')
		value, ok := values[name]
		if !ok {
			return nil, fmt.Errorf("missing projected PR field %q", name)
		}
		b.Write(value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}
