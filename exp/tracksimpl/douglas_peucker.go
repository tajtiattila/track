package tracksimpl

import (
	"time"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/trackmath"
)

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

	Adaptive bool

	maxDepth int
}

func (f EndPointFit) Run(dst, src track.Track) track.Track {
	n := len(src)
	if n <= 2 {
		return append(dst, src...)
	}

	last := src[n-1]

	if f.Window <= 16 {
		f.Window = n
	}

	if f.Adaptive {
		f.maxDepth = 1
		for m := 1; m < n; m *= 2 {
			f.maxDepth++
		}
		f.maxDepth *= 2
	}

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

	var o int
	if f.Adaptive && n > adaptiveWin && depth > f.maxDepth {
		o = n / 4
	} else if n >= f.Window {
		o = f.Window / 4
	}

	if o != 0 {
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

type point4 [4]float64

func pt4(p trackmath.Point3, t time.Time) point4 {
	tv := float64(t.UnixNano()/1e6) / 1e3
	return point4{p[0], p[1], p[2], tv}
}

func (a point4) Add(b point4) point4 {
	return point4{
		a[0] + b[0],
		a[1] + b[1],
		a[2] + b[2],
		a[3] + b[3],
	}
}

func (a point4) Sub(b point4) point4 {
	return point4{
		a[0] - b[0],
		a[1] - b[1],
		a[2] - b[2],
		a[3] - b[3],
	}
}

func (a point4) Mul(s float64) point4 {
	return point4{
		a[0] * s,
		a[1] * s,
		a[2] * s,
		a[3] * s,
	}
}

func (a point4) Dot(b point4) float64 {
	return (a[0]*b[0] +
		a[1]*b[1] +
		a[2]*b[2] +
		a[3]*b[3])
}

type line4 struct {
	a, b point4
	v    point4 // a→b vector
}

func l4(a, b point4) line4 {
	v := b.Sub(a)
	vv := v.Dot(v)
	return line4{
		a: a,
		v: v.Mul(1 / vv),
	}
}

func (l *line4) distsq(p point4) float64 {
	w := p.Sub(l.a)
	c := l.v.Dot(w)
	var q point4
	switch {
	case c <= 0:
		q = w
	case 1 <= c:
		q = p.Sub(l.b)
	default:
		lp := l.a.Add(l.v.Mul(c))
		q = p.Sub(lp)
	}
	return q.Dot(q)
}
