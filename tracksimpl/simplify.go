// Package tracksimpl implements GPS track simplification algorithms.
//
// Algorithms of this package are well known polyline simplification
// algorithms modified to take special attention to the time parameter.
//
// Algorithms assume that input tracks are in chronological order,
// and return tracks that are also in chronological order.
//
// When an algorithm has a maximum distance parameter d,
// a time lookup such as track.Track.At on the simplified track
// will yield a position within d meters of the original position,
// unless noted otherwise.
package tracksimpl

import (
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/geomath"
)

// Algorithm is a track simplification algorithm.
type Algorithm interface {
	Run(dst, src track.Track) track.Track
}

// Run applies the track simplification algorithms to src.
//
// It appends points to dst and returns the result slice.
func Run(dst, src track.Track, algo ...Algorithm) track.Track {
	ofs := len(dst)
	for _, a := range algo {
		dst = a.Run(dst[:ofs], src)
		src = dst[ofs:]
	}
	return dst
}

func pt3(p track.Point) geomath.Point3 {
	return geomath.Pt3(p.Lat(), p.Long())
}

func dist3sq(a, b geomath.Point3) float64 {
	d := a.Sub(b)
	return d.Dot(d)
}
