package track

import (
	"sort"
	"time"

	"github.com/tajtiattila/track/trackutil"
)

// Len returns the number of track points in trk.
func (trk Track) Len() int { return len(trk) }

// Pt returns the time stamp and position at index i.
func (trk Track) Pt(i int) (t time.Time, lat, long float64) {
	p := trk[i]
	t, lat, long = p.Time(), p.Lat(), p.Long()
	return
}

// StartTime returns the time of the first point in trk.
//
// It returns the zero time if trk is empty.
func (trk Track) StartTime() time.Time {
	if len(trk) == 0 {
		return time.Time{}
	}

	return trk[0].Time()
}

// EndTime returns the time of the last point in trk.
//
// It returns the zero time if trk is empty.
func (trk Track) EndTime() time.Time {
	if len(trk) == 0 {
		return time.Time{}
	}

	return trk[len(trk)-1].Time()
}

// HasTime checks if trk has a position for the given time.
func (trk Track) HasTime(t time.Time) bool {
	if len(trk) == 0 {
		return false
	}

	tt := itime(t)
	if p := trk[0]; tt < p.t {
		return false
	}
	if p := trk[len(trk)-1]; tt > p.t {
		return false
	}

	return true
}

// TimeIndex returns the index i of the first track point in trk
// where trk[i].Time().Before(t) is false.
//
// It returns len(trk) if t is after the time of the last track point.
func (trk Track) TimeIndex(t time.Time) int {
	tt := itime(t)
	return sort.Search(len(trk), func(i int) bool {
		return tt < trk[i].t
	})
}

// At calculates the interpolated lat and long
// of trk for the given time.
//
// It returns the closest point if trk.HasTime(t) is false,
// and (0, 0) if trk is empty.
func (trk Track) At(t time.Time) (lat, long float64) {
	return trackutil.Lookup(trk, t)
}
