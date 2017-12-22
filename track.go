// Package track is a simple GPS track parser in Go.
//
// It supports GPX, KML and Google location history JSON formats.
package track

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// Point represents a track point.
type Point struct {
	T         time.Time // timestamp of track point
	Lat, Long float64   // geographical coordinates (WGS 84)
}

// Track is a series of track points.
type Track []Point

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("track: unknown format")

// DecodeError indicates that a format was recognised but
// the decoding failed.
type DecodeError struct {
	Reason string
}

func (e *DecodeError) Error() string { return e.Reason }

func decodeError(format string, a ...interface{}) error {
	return &DecodeError{fmt.Sprintf(format, a...)}
}

// errBadFormat is returned by decoders to indicate
// the underlying encoding doesn't match the decoders own format
var errBadFormat = errors.New("track: bad format")

// Decode decodes GPX, KML and Google location history JSON formats.
//
// It returns track points found in chronological order.
//
// It returns ErrFormat if the format is not recognised,
// or the encoded data doesn't contain any tracks.
func Decode(r io.Reader) (Track, error) {
	var buf bytes.Buffer
	tr := io.TeeReader(io.LimitReader(r, 1<<20), &buf)
	cr := &countReader{r: r}

	for _, f := range formats {
		rx := io.MultiReader(bytes.NewReader(buf.Bytes()), tr, cr)
		t, err := f(rx)
		if len(t) != 0 && err == nil {
			sort.Sort(byTime(t))
			return t, nil
		}
		if _, ok := err.(*DecodeError); ok {
			return nil, err
		}
		if cr.n != 0 {
			break // read past prefix, can't try other formats
		}
	}
	return nil, ErrFormat
}

type formatFunc func(io.Reader) (Track, error)

var formats = []formatFunc{
	decodeGPX,
	decodeKML,
	decodeGoogleJSON,
}

type countReader struct {
	r io.Reader
	n int64
}

func (cr *countReader) Read(p []byte) (n int, err error) {
	n, err = cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

type byTime []Point

func (t byTime) Len() int           { return len(t) }
func (t byTime) Less(i, j int) bool { return t[i].T.Before(t[j].T) }
func (t byTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
