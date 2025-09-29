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

// decodeJSON decodes a single JSON value from r into v.
// Unknown fields are allowed to accommodate daemon schema drift.
func decodeJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(v)
}

// encodeJSON returns an io.Reader with the JSON encoding of v.
func encodeJSON(v any) (io.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// DecodeStream calls fn once per JSON object in r. Stops on EOF or ctx cancel.
func DecodeStream(ctx context.Context, r io.Reader, fn func(json.RawMessage) error) error {
	dec := json.NewDecoder(r)
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

// readBodyBestEffort reads and closes r up to max bytes.
func readBodyBestEffort(r io.ReadCloser, max int64) ([]byte, error) {
	defer r.Close()
	return ioutil.ReadAll(io.LimitReader(r, max))
}
