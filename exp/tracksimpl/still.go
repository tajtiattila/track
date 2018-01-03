package tracksimpl

import (
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/trackmath"
)

// StillFilter filters src by dropping track points
// that are within maxd meters.
//
// It appends points to dst and returns the result slice.
//
// It replaces all subslices of src[m:n] where
// src[i] for m <= i <= n are closer to src[m] than maxd
// with the two points src[m] and src[n] in the output.
func StillFilter(maxd float64) Algorithm {
	return &stillFilter{dd: maxd * maxd}
}

type stillFilter struct {
	dd float64 // meters squared
}

func (f *stillFilter) run(dst, src track.Track) track.Track {
	var stop3 trackmath.Point3
	var last track.Point
	emitLast := false
	for i, p := range src {
		p3 := pt3(p)

		if i == 0 {
			stop3 = p3
			dst = append(dst, p)
		} else {
			if dist3sq(stop3, p3) > f.dd {
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
