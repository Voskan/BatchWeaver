package diagnostics

import (
	"encoding/json"
	"io"
)

// JSONSchemaVersion is the version of the machine-readable diagnostic document
// produced by JSONFormatter. It is independent of any configuration schema
// version and is incremented only when the JSON shape changes.
const JSONSchemaVersion = 1

// JSONFormatter renders diagnostics as a deterministic JSON document suitable
// for editors, tests, and future language-server integration.
type JSONFormatter struct{}

// NewJSONFormatter returns a JSONFormatter.
func NewJSONFormatter() JSONFormatter { return JSONFormatter{} }

// documentJSON is the top-level JSON shape.
type documentJSON struct {
	SchemaVersion int              `json:"schema_version"`
	Diagnostics   []diagnosticJSON `json:"diagnostics"`
}

type diagnosticJSON struct {
	Code     string        `json:"code"`
	Severity string        `json:"severity"`
	Message  string        `json:"message"`
	Source   string        `json:"source,omitempty"`
	Range    *rangeJSON    `json:"range,omitempty"`
	Details  string        `json:"details,omitempty"`
	Related  []relatedJSON `json:"related,omitempty"`
	Fixes    []fixJSON     `json:"fixes,omitempty"`
}

type relatedJSON struct {
	Message string     `json:"message"`
	Range   *rangeJSON `json:"range,omitempty"`
}

type fixJSON struct {
	Message string     `json:"message"`
	Edits   []editJSON `json:"edits,omitempty"`
}

type editJSON struct {
	Range   rangeJSON `json:"range"`
	NewText string    `json:"new_text"`
}

type rangeJSON struct {
	Start positionJSON  `json:"start"`
	End   *positionJSON `json:"end,omitempty"`
}

type positionJSON struct {
	File   string `json:"file,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

// Format writes the collection to w as a JSON document. Diagnostics are sorted
// deterministically, the document ends with a newline, and HTML escaping is
// disabled so that characters such as quotes are rendered readably while
// remaining valid JSON.
func (f JSONFormatter) Format(w io.Writer, c Collection) error {
	sorted := c.Sorted()
	doc := documentJSON{
		SchemaVersion: JSONSchemaVersion,
		Diagnostics:   make([]diagnosticJSON, 0, sorted.Len()),
	}
	for _, d := range sorted.diags {
		doc.Diagnostics = append(doc.Diagnostics, toDiagnosticJSON(d))
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func toDiagnosticJSON(d Diagnostic) diagnosticJSON {
	out := diagnosticJSON{
		Code:     string(d.Code),
		Severity: d.Severity.String(),
		Message:  d.Message,
		Source:   d.Source,
		Range:    toRangeJSON(d.Range),
		Details:  d.Details,
	}
	for _, rel := range d.Related {
		out.Related = append(out.Related, relatedJSON{
			Message: rel.Message,
			Range:   toRangeJSON(rel.Range),
		})
	}
	for _, fix := range d.Fixes {
		fj := fixJSON{Message: fix.Message}
		for _, e := range fix.Edits {
			fj.Edits = append(fj.Edits, editJSON{
				Range:   rangeJSON{Start: toPositionJSON(e.Range.Start), End: endPositionJSON(e.Range.End)},
				NewText: e.NewText,
			})
		}
		out.Fixes = append(out.Fixes, fj)
	}
	return out
}

// toRangeJSON returns nil for a zero range so it is omitted from output.
func toRangeJSON(r Range) *rangeJSON {
	if r.IsZero() {
		return nil
	}
	return &rangeJSON{
		Start: toPositionJSON(r.Start),
		End:   endPositionJSON(r.End),
	}
}

// endPositionJSON returns nil for a zero end position so it is omitted.
func endPositionJSON(p Position) *positionJSON {
	if p.IsZero() {
		return nil
	}
	pj := toPositionJSON(p)
	return &pj
}

func toPositionJSON(p Position) positionJSON {
	return positionJSON(p)
}
