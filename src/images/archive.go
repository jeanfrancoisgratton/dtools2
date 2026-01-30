// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30 07:59
// Original filename: src/images/archive.go

package images

import (
	std_bzip2 "compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	ds_bzip2 "github.com/dsnet/compress/bzip2"
	"github.com/ulikunitz/xz"
)

func (r *readerWithClose) Close() error {
	if r.closeFn == nil {
		return nil
	}
	return r.closeFn()
}

type writerWithClose struct {
	io.Writer
	closeFn func() error
}

func (w *writerWithClose) Close() error {
	if w.closeFn == nil {
		return nil
	}
	return w.closeFn()
}

func detectCompression(filename string) archiveCompression {
	name := strings.ToLower(filename)

	switch {
	case strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".gz"):
		return compGzip
	case strings.HasSuffix(name, ".tar.bz2") || strings.HasSuffix(name, ".tbz2") || strings.HasSuffix(name, ".tbz") || strings.HasSuffix(name, ".bz2"):
		return compBzip2
	case strings.HasSuffix(name, ".tar.xz") || strings.HasSuffix(name, ".txz") || strings.HasSuffix(name, ".xz"):
		return compXz
	default:
		return compNone
	}
}

func openArchiveReader(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	switch detectCompression(filename) {
	case compNone:
		return f, nil

	case compGzip:
		zr, err := gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &readerWithClose{
			Reader: zr,
			closeFn: func() error {
				err1 := zr.Close()
				err2 := f.Close()
				if err1 != nil {
					return err1
				}
				return err2
			},
		}, nil

	case compBzip2:
		br := std_bzip2.NewReader(f)
		return &readerWithClose{
			Reader: br,
			closeFn: func() error {
				return f.Close()
			},
		}, nil

	case compXz:
		xr, err := xz.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &readerWithClose{
			Reader: xr,
			closeFn: func() error {
				return f.Close()
			},
		}, nil

	default:
		_ = f.Close()
		return nil, fmt.Errorf("unsupported archive format: %s", filename)
	}
}

func openArchiveWriter(filename string) (io.WriteCloser, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	switch detectCompression(filename) {
	case compNone:
		return f, nil

	case compGzip:
		zw := gzip.NewWriter(f)
		return &writerWithClose{
			Writer: zw,
			closeFn: func() error {
				err1 := zw.Close()
				err2 := f.Close()
				if err1 != nil {
					return err1
				}
				return err2
			},
		}, nil

	case compBzip2:
		zw, err := ds_bzip2.NewWriter(f, &ds_bzip2.WriterConfig{Level: ds_bzip2.DefaultCompression})
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &writerWithClose{
			Writer: zw,
			closeFn: func() error {
				err1 := zw.Close()
				err2 := f.Close()
				if err1 != nil {
					return err1
				}
				return err2
			},
		}, nil

	case compXz:
		_ = f.Close()
		return nil, fmt.Errorf("xz compression is not supported for save (output: %s)", filename)

	default:
		_ = f.Close()
		return nil, fmt.Errorf("unsupported archive format: %s", filename)
	}
}
