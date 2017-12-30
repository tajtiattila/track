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

const coordUnit = 1e7

// Point represents a track point.
//
// It has millisecond time precision.
//
// Geographical coordinates are integers of
// the degree value multiplied by 1e7,
// therefore have a precision of at least 0.0111 meters.
type Point struct {
	t         int64 // milliseconds since January 1, 1970 UTC
	lat, long int32 // values multiplied by 1e7
}

// Pt returns a new track point.
func Pt(t time.Time, lat, long float64) Point {
	return Point{
		t:    t.UnixNano() / 1e6,
		lat:  icoord(lat),
		long: icoord(long),
	}
}

func icoord(v float64) int32 {
	var d float64
	if v > 0 {
		d = 0.5
	} else {
		d = -0.5
	}
	return int32(v*coordUnit + d)
}

// Time returns the time of p.
func (p Point) Time() time.Time { return time.Unix(p.t/1000, (p.t%1000)*1e6).UTC() }

// Lat returns the geographical latitude of p.
func (p Point) Lat() float64 { return float64(p.lat) / coordUnit }

// Long returns the geographical longitude of p.
func (p Point) Long() float64 { return float64(p.long) / coordUnit }

// Track is a series of track points.
type Track []Point

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("track: unknown format")

// DecodeError indicates that a format was recognised but decoding failed.
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
			fmt.Println("hopp")
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
func (t byTime) Less(i, j int) bool { return t[i].t < t[j].t }
func (t byTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
