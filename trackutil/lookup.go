package trackutil

import "time"

type Track interface {
	Len() int
	TimeIndex(t time.Time) int
	Pt(i int) (t time.Time, lat, long float64)
}

func Lookup(trk Track, t time.Time) (lat, long float64) {
	i := trk.TimeIndex(t)

	if i == 0 {
		if trk.Len() == 0 {
			return 0, 0
		}
		_, lat, long := trk.Pt(0)
		return lat, long
	} else if n := trk.Len(); i == n {
		_, lat, long := trk.Pt(n - 1)
		return lat, long
	}

	pt, plat, plong := trk.Pt(i - 1)
	qt, qlat, qlong := trk.Pt(i)

	pd := float64(t.Sub(pt))
	qd := float64(qt.Sub(t))
	m := pd + qd
	pw := qd / m
	qw := pd / m

	lat = pw*plat + qw*qlat

	px, qx := plong, qlong
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
