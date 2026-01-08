// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 19:32
// Original filename: src/extras/outputhandler.go

package extras

import (
	"encoding/json"
	"os"
	"path/filepath"
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
