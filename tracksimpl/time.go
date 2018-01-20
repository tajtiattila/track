package tracksimpl

import (
	"time"

	"github.com/tajtiattila/track"
)

// RoundTime rounds all time values to Dt in src,
// and replaces multiple entries with the same time value
// with a single point having the average location of those points.
type RoundTime struct {
	Dt time.Duration
}

func (tt RoundTime) Run(dst, src track.Track) track.Track {
	if len(src) == 0 {
		return dst
	}

	n := len(src) - 1

	f := timeFilter{
		dt:  tt.Dt,
		dst: dst,
	}

	f.start(src[0])
	for _, b := range src[1:n] {
		f.next(b)
	}
	f.flush()

	return f.dst
}

type timeFilter struct {
	dst track.Track
	dt  time.Duration

	a  track.Point
	at time.Time

	sumlat, sumlong float64
	sumn            int
}

func (f *timeFilter) start(a track.Point) {
	f.a, f.at = a, a.Time().Round(f.dt)
	f.sumn = 0
}

func (f *timeFilter) next(b track.Point) {
	bt := b.Time().Round(f.dt)
	if f.at == bt {
		if f.sumn == 0 {
			f.sumlat, f.sumlong = f.a.Lat(), f.a.Long()
			f.sumn++
		}
		f.sumlat += b.Lat()
		f.sumlong += b.Long()
		f.sumn++
	} else {
		f.flush()
		f.a, f.at = b, bt
		f.sumn = 0
	}
}

func (f *timeFilter) flush() {
	if f.sumn > 0 {
		m := 1 / float64(f.sumn)
		f.dst = append(f.dst, track.Pt(f.at, f.sumlat/m, f.sumlong/m))
	} else {
		f.dst = append(f.dst, f.a)
	}
}
