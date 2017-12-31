package track

import (
	"sort"
	"time"
)

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
	i := trk.TimeIndex(t)

	if i == 0 {
		if len(trk) == 0 {
			return 0, 0
		}
		p := trk[0]
		return p.Lat(), p.Long()
	} else if i == len(trk) {
		p := trk[len(trk)-1]
		return p.Lat(), p.Long()
	}

	p, q := trk[i-1], trk[i]
	pd := float64(t.Sub(p.Time()))
	qd := float64(q.Time().Sub(t))
	m := pd + qd
	pw := qd / m
	qw := pd / m

	lat = pw*p.Lat() + qw*q.Lat()

	px, qx := p.Long(), q.Long()
	if qx < px {
		pw, qw = qw, pw
		px, qx = qx, px
	}

	if px < -90 && 90 < qx {
		// track segment passing over date (±180° latitude) line
		px += 360
		long = pw*px + qw*qx
		if long > 180 {
			long -= 360
		}
	} else {
		long = pw*px + qw*qx
	}

	return lat, long
}
