// Package track implements GPS track functions.
package track

import (
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

// Sort sorts points of trk in chronological order.
func (trk Track) Sort() {
	sort.Sort(byTime(trk))
}

type byTime []Point

func (t byTime) Len() int           { return len(t) }
func (t byTime) Less(i, j int) bool { return t[i].t < t[j].t }
func (t byTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
