package tracksimpl

import (
	"github.com/tajtiattila/track"
)

// EndPointFit implements a variant of
// the Ramer–Douglas–Peucker iterative end-point fit algorithm,
// also known as the Douglas–Peucker algorithm
// that provides an excellent approximation of the original track.
//
// The algorithm is modified to avoid the worst case complexity.
//
// It provides good results but has an average comlexity of O(n·log n).
// If Full is specified, it has a worst case complexity of O(n²).
type EndPointFit struct {
	D float64 // maximum error distance in meters

	Full bool // run full algorithm

	maxDepth int
}

func (f EndPointFit) Run(dst, src track.Track) track.Track {
	n := len(src)
	if n <= 2 {
		return append(dst, src...)
	}

	last := src[n-1]

	f.maxDepth = 1
	for m := 1; m < n; m *= 2 {
		f.maxDepth++
	}
	f.maxDepth *= 2

	return append(f.run(0, dst, src), last)
}

// run performs the iterative end-point fit algorithm,
// but does not append the last point of src to dst.
func (f EndPointFit) run(depth int, dst, src track.Track) track.Track {
	n := len(src) - 1
	if n < 2 {
		return append(dst, src[:n]...)
	}

	a, b := src[0], src[n]
	a3, b3 := pt3(a), pt3(b)

	dt := float64(b.Time().Sub(a.Time())) // nanoseconds
	if dt < 1 {
		return append(dst, src[:n]...)
	}
	v := b3.Sub(a3).Muls(1 / dt) // meters/nanosecond

	imax := 1
	var dmax float64
	for i := 1; i < n; i++ {
		p := src[i]
		p3 := pt3(p)

		dt = float64(p.Time().Sub(a.Time()))
		q3 := a3.Add(v.Muls(dt))

		if d := dist3sq(p3, q3); d > dmax {
			imax, dmax = i, d
		}
	}

	if dmax <= f.D*f.D {
		return append(dst, a)
	}

	const adaptiveWin = 32

	if !f.Full && n > adaptiveWin && depth > f.maxDepth {
		o := n / 4
		m := n / 2
		if imax < o {
			dst = f.run(depth+1, dst, src[:imax+1])
			dst = f.run(depth+1, dst, src[imax:m+1])
			return f.run(depth+1, dst, src[m:])
		} else if imax+o > n {
			dst = f.run(depth+1, dst, src[:m+1])
			dst = f.run(depth+1, dst, src[m:imax+1])
			return f.run(depth+1, dst, src[imax:])
		}
	}

	dst = f.run(depth+1, dst, src[:imax+1])
	return f.run(depth+1, dst, src[imax:])
}
