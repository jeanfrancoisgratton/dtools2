// extras/outputhandler.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 19:32
// Original filename: src/extras/outputhandler.go

package extras

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// MarshalJSON marshals the provided payload into an indented JSON byte slice.
// A trailing newline is always appended.
func MarshalJSON(payload any) ([]byte, *ce.CustomError) {
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to marshal JSON", Message: err.Error()}
	}

	// Ensure trailing newline.
	if len(b) == 0 || b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	return b, nil
}

// PrintJSONBytes prints JSON bytes to stdout. This expects valid JSON.
// (If you want jq-like coloring, swap this implementation to call your hfjson.Print().)
func PrintJSONBytes(jsonBytes []byte) *ce.CustomError {
	if _, err := os.Stdout.Write(jsonBytes); err != nil {
		return &ce.CustomError{Title: "Unable to write JSON output", Message: err.Error()}
	}
	return nil
}

// WriteFileAtomic writes data to outputFile using a temp file + rename.
// This avoids truncating an existing output file if the write fails.
func WriteFileAtomic(outputFile string, data []byte) *ce.CustomError {
	if strings.TrimSpace(outputFile) == "" {
		return &ce.CustomError{Title: "Invalid output file", Message: "outputFile is empty"}
	}

	dir := filepath.Dir(outputFile)
	base := filepath.Base(outputFile)

	tmp, err := os.CreateTemp(dir, "."+base+".tmp.*")
	if err != nil {
		return &ce.CustomError{Title: "Unable to create temporary file", Message: err.Error()}
	}
	tmpName := tmp.Name()

	// Cleanup temp on failure.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return &ce.CustomError{Title: "Unable to write output file", Message: err.Error()}
	}

	if err := tmp.Chmod(0o644); err != nil {
		return &ce.CustomError{Title: "Unable to set output file mode", Message: err.Error()}
	}

	if err := tmp.Close(); err != nil {
		return &ce.CustomError{Title: "Unable to close output file", Message: err.Error()}
	}

	// os.Rename is atomic on POSIX when source and destination are on the same filesystem.
	if err := os.Rename(tmpName, outputFile); err != nil {
		return &ce.CustomError{Title: "Unable to finalize output file", Message: err.Error()}
	}

	return nil
}

// Send2File marshals payload to JSON, writes it to outputFile, and returns the JSON bytes.
// This is meant to be used by list commands so the same marshaled bytes can be reused for
// stdout JSON output (when requested) without double-marshalling.
func Send2File(payload any, outputFile string) ([]byte, *ce.CustomError) {
	b, cerr := MarshalJSON(payload)
	if cerr != nil {
		return nil, cerr
	}

	if cerr := WriteFileAtomic(outputFile, b); cerr != nil {
		return nil, cerr
	}

	return b, nil
}

// -----------------------------------------------------------------------------
// --format support (plaintext output)
// -----------------------------------------------------------------------------

// ExtractFormatRows returns plaintext rows for a slice/array of structs.
//
// format accepts:
//   - "Name" (single field)
//   - "Name,Id" (multiple fields; output is tab-separated per row)
//   - docker-ish single token like "{{.Name}}" or ".Name" (normalized to "Name")
//   - simple dotted paths like "IPAM.Driver" (best-effort)
func ExtractFormatRows(list any, format string) ([][]string, *ce.CustomError) {
	keys := parseFormatKeys(format)
	if len(keys) == 0 {
		return nil, nil
	}

	v := reflect.ValueOf(list)
	if !v.IsValid() {
		return nil, &ce.CustomError{Title: "Invalid list", Message: "nil value"}
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, &ce.CustomError{Title: "Invalid list", Message: "nil pointer"}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, &ce.CustomError{Title: "Invalid list", Message: "expected slice or array"}
	}

	rows := make([][]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		e := v.Index(i)
		if e.Kind() == reflect.Pointer {
			if e.IsNil() {
				continue
			}
			e = e.Elem()
		}
		if e.Kind() != reflect.Struct {
			return nil, &ce.CustomError{Title: "Invalid element", Message: "expected struct elements"}
		}

		row := make([]string, 0, len(keys))
		for _, k := range keys {
			val, ok := getPathValue(e, strings.Split(k, "."))
			if !ok {
				// Match docker behavior loosely: missing field => empty string
				row = append(row, "")
				continue
			}
			row = append(row, valueToString(e, k, val))
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// PrintFormatRows writes plaintext rows to stdout.
// Each row is tab-separated; each row ends with a newline.
func PrintFormatRows(rows [][]string) *ce.CustomError {
	for _, r := range rows {
		if _, err := fmt.Fprintln(os.Stdout, strings.Join(r, "\t")); err != nil {
			return &ce.CustomError{Title: "Unable to write formatted output", Message: err.Error()}
		}
	}
	return nil
}

func parseFormatKeys(format string) []string {
	f := strings.TrimSpace(format)
	if f == "" {
		return nil
	}

	// If user copy-pastes docker style "{{.Name}}", normalize it.
	if strings.Contains(f, "{{") && strings.Contains(f, "}}") && !strings.Contains(f, ",") {
		f = normalizeFormatToken(f)
		if f == "" {
			return nil
		}
		return []string{f}
	}

	parts := strings.Split(f, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = normalizeFormatToken(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func normalizeFormatToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Strip docker-ish wrappers.
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		s = strings.TrimPrefix(s, "{{")
		s = strings.TrimSuffix(s, "}}")
		s = strings.TrimSpace(s)
	}

	// Strip leading '.' if present (".Name" -> "Name").
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, ".") {
		s = strings.TrimPrefix(s, ".")
	}

	// If there's still spaces (templates), keep only the first token.
	if idx := strings.IndexAny(s, " \t"); idx != -1 {
		s = s[:idx]
	}

	return strings.TrimSpace(s)
}

func getPathValue(v reflect.Value, path []string) (reflect.Value, bool) {
	cur := v
	for _, seg := range path {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return reflect.Value{}, false
		}

		for cur.Kind() == reflect.Pointer {
			if cur.IsNil() {
				return reflect.Value{}, false
			}
			cur = cur.Elem()
		}

		switch cur.Kind() {
		case reflect.Struct:
			fv, ok := findStructFieldByNameOrJSON(cur, seg)
			if !ok {
				return reflect.Value{}, false
			}
			cur = fv
		case reflect.Map:
			// Only support string keys.
			if cur.Type().Key().Kind() != reflect.String {
				return reflect.Value{}, false
			}
			mv := cur.MapIndex(reflect.ValueOf(seg))
			if !mv.IsValid() {
				return reflect.Value{}, false
			}
			cur = mv
		default:
			return reflect.Value{}, false
		}
	}

	return cur, true
}

func findStructFieldByNameOrJSON(v reflect.Value, key string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// Only exported fields.
		if f.PkgPath != "" {
			continue
		}

		if strings.EqualFold(f.Name, key) {
			return v.Field(i), true
		}

		jt := f.Tag.Get("json")
		if jt != "" {
			if idx := strings.IndexByte(jt, ','); idx != -1 {
				jt = jt[:idx]
			}
			if jt != "" && jt != "-" && strings.EqualFold(jt, key) {
				return v.Field(i), true
			}
		}
	}
	return reflect.Value{}, false
}

func valueToString(parent reflect.Value, key string, v reflect.Value) string {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// Special-case container names to match docker output expectations.
	if parent.IsValid() && parent.Kind() == reflect.Struct {
		if parent.Type().Name() == "ContainerSummary" {
			if strings.EqualFold(key, "Name") || strings.EqualFold(key, "Names") {
				if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.String {
					if v.Len() == 0 {
						return ""
					}
					s := v.Index(0).String()
					return strings.TrimPrefix(s, "/")
				}
			}
		}
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", v.Float())
	case reflect.Slice, reflect.Array:
		// Best effort: if []string, join with comma. Otherwise JSON encode.
		if v.Type().Elem().Kind() == reflect.String {
			ss := make([]string, 0, v.Len())
			for i := 0; i < v.Len(); i++ {
				ss = append(ss, v.Index(i).String())
			}
			return strings.Join(ss, ",")
		}
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return ""
		}
		return string(b)
	case reflect.Struct, reflect.Map:
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return ""
		}
		return string(b)
	default:
		return fmt.Sprint(v.Interface())
	}
}
