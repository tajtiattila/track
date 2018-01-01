// Package track is a simple GPS track parser in Go.
//
// It supports GPX, KML and Google location history JSON formats.
package track

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
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
	if lat < -90 {
		lat = -90
	} else if lat > 90 {
		lat = 90
	}

	if long < -180 || long >= 180 {
		long = math.Mod(long, 360)
		if long < 180 {
			long += 360
		} else if long >= 180 {
			long -= 360
		}
	}

	return Point{
		t:    itime(t),
		lat:  icoord(lat),
		long: icoord(long),
	}
}

func itime(t time.Time) int64 { return t.UnixNano() / 1e6 }

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

// Lat returns the geographical latitude of p in degrees.
func (p Point) Lat() float64 { return float64(p.lat) / coordUnit }

// Long returns the geographical longitude of p in degrees.
func (p Point) Long() float64 { return float64(p.long) / coordUnit }

// Track is a series of track points
// in chronological order.
type Track []Point

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("track: unknown format")

// DecodeError indicates that a format was recognised but decoding failed.
type DecodeError struct {
	Reason error
}

func (e *DecodeError) Error() string { return "track: " + e.Reason.Error() }

func decodeError(format string, a ...interface{}) error {
	return &DecodeError{fmt.Errorf(format, a...)}
}

// errBadFormat is returned by decoders to indicate
// the underlying encoding doesn't match the decoders own format
var errBadFormat = errors.New("track: bad format")

// DetectFormat determines if data represents a track file.
//
// The string returned the name of the format.
//
// Data should be large enough to contain the
// first data element (such as the document node in XML).
// Typically the first few kilobytes is sufficient.
func DetectFormat(data []byte) (string, bool) {
	for _, f := range formats {
		if f.detect(data) {
			return f.name, true
		}
	}
	return "", false
}

// Decode decodes GPX, KML and Google location history JSON formats.
//
// It returns track points found in chronological order.
//
// It returns ErrFormat if the format is not recognised,
// or the encoded data doesn't contain any tracks.
func Decode(r io.Reader) (Track, error) {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, io.LimitReader(r, 64<<10))
	if err != nil {
		return nil, err
	}

	var trk Track
	for _, f := range formats {
		if f.detect(buf.Bytes()) {
			trk, err = f.decode(io.MultiReader(buf, r))
			break
		}
	}

	if err != nil {
		return nil, err
	}

	if len(trk) == 0 {
		return nil, ErrFormat
	}

	trk.Sort()
	return trk, nil
}

type detectFunc func(p []byte) bool
type decodeFunc func(r io.Reader) (Track, error)

type format struct {
	name   string
	detect detectFunc
	decode decodeFunc
}

var formats []format

func registerFormat(
	name string,
	detect detectFunc,
	decode decodeFunc,
) {

	for _, f := range formats {
		if f.name == name {
			panic("name already registered")
		}
	}
	formats = append(formats, format{name, detect, decode})
}

// Sort sorts points of trk in chronological order.
func (trk Track) Sort() {
	sort.Sort(byTime(trk))
}

type byTime []Point

func (t byTime) Len() int           { return len(t) }
func (t byTime) Less(i, j int) bool { return t[i].t < t[j].t }
func (t byTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
