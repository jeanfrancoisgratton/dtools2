// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 09:20
// Original filename: src/rest/json.go

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
)

// decodeJSON decodes a single JSON object from r into v.
// Unknown fields are rejected to surface server/schema drift early.
func decodeJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// encodeJSON returns an io.Reader for the JSON encoding of v.
// Useful for request bodies without pulling in extra deps.
func encodeJSON(v any) (io.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// DecodeStream calls fn once per JSON object found in r.
// Works for newline-delimited JSON streams and for concatenated objects.
// Stops on ctx cancellation or EOF. Returns first non-EOF error.
func DecodeStream(ctx context.Context, r io.Reader, fn func(json.RawMessage) error) error {
	dec := json.NewDecoder(r)
	// Allow numbers without automatic float64 conversion if callers care.
	dec.UseNumber()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := fn(raw); err != nil {
			return err
		}
	}
}

// readBodyBestEffort reads and closes resp.Body up to a sane limit.
// Use to surface server error payloads in errors.
func readBodyBestEffort(r io.ReadCloser, max int64) ([]byte, error) {
	defer r.Close()
	limited := io.LimitReader(r, max)
	return ioutil.ReadAll(limited)
}
