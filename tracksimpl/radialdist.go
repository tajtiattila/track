package tracksimpl

import (
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/geomath"
)

// RadialDistance filters src by dropping track points
// that are within D meters.
//
// It appends points to dst and returns the result slice.
//
// It replaces all subslices of src[m:n] where
// src[i] for m <= i <= n are closer to src[m] than D
// with the two points src[m] and src[n] in the output.
type RadialDistance struct {
	D float64 // maximum distance in meters
}

func (rd RadialDistance) Run(dst, src track.Track) track.Track {
	dd := rd.D / 2
	dd *= dd
	var stop3 geomath.Point3
	var last track.Point
	emitLast := false
	for i, p := range src {
		p3 := pt3(p)

		if i == 0 {
			stop3 = p3
			dst = append(dst, p)
		} else {
			if dist3sq(stop3, p3) > dd {
				if emitLast {
					dst = append(dst, last)
				}
				stop3 = p3
				dst = append(dst, p)
				emitLast = false
			} else {
				emitLast = true
			}
		}

		last = p
	}
	if emitLast {
		dst = append(dst, last)
	}
	return dst
}

func findStops(src track.Track, d float64, f func(i int, stopped bool)) {
	n := len(src)
	if n == 0 {
		return
	}

	dd := d * d

	// stop point
	q := src[0]
	q3 := pt3(q)

	stopped := false
	f(0, stopped)

	for i := 1; i < n; i++ {
		p := src[i]
		p3 := pt3(p)

		s := dist3sq(p3, q3) <= dd
		if s != stopped {
			stopped = s
			f(i, stopped)
		}

		if !s {
			q, q3 = p, p3
		}
	}
}
