package tracksimpl

import "github.com/tajtiattila/track"

// EndPointFit implements
// the Ramer–Douglas–Peucker iterative end-point fit algorithm,
// also known as the Douglas–Peucker algorithm
// using D as the maximum distance.
//
// When Window is not zero, the algorithm is executed only
// at most on the specified number of points. This should
// improve worst case performance at the expense of slightly
// less efficient result tracks.
//
// It provides good results but has an average comlexity of O(n·log n)
// and a worst case complexity of O(n²).
type EndPointFit struct {
	D float64 // maximum distance in meters

	Window int // maximum number of points to process at once
}

func (f EndPointFit) Run(dst, src track.Track) track.Track {
	n := len(src)
	if n <= 2 {
		return append(dst, src...)
	}

	last := src[n-1]

	if f.Window <= 0 {
		return append(f.run(dst, src), last)
	}

	for i := 0; i < n; {
		j := i + f.Window
		if j < n {
			dst = f.run(dst, src[i:j+1])
			i = j
		} else {
			dst = f.run(dst, src[i:])
			i = n
		}
	}

	return append(dst, last)
}

// run performs the iterative end-point fit algorithm,
// but does not append the last point of src to dst.
func (f EndPointFit) run(dst, src track.Track) track.Track {
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

	dst = f.run(dst, src[:imax+1])
	return f.run(dst, src[imax:])
}
