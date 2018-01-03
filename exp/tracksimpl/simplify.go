// Package tracksimpl implements experimental GPS track simplification algorithms.
package tracksimpl

import (
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/trackmath"
)

// Algorithm is a track simplification algorithm.
type Algorithm interface {
	run(dst, src track.Track) track.Track
}

// Run applies the track simplification algorithms to src.
//
// It appends points to dst and returns the result slice.
func Run(dst, src track.Track, algo ...Algorithm) track.Track {
	ofs := len(dst)
	for _, a := range algo {
		dst = a.run(dst[:ofs], src)
		src = dst[ofs:]
	}
	return dst
}

func pt3(p track.Point) trackmath.Point3 {
	return trackmath.Pt3(p.Lat(), p.Long())
}

func dist3sq(a, b trackmath.Point3) float64 {
	d := a.Sub(b)
	return d.Dot(d)
}
