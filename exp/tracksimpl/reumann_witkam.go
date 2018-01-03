package tracksimpl

import (
	"github.com/tajtiattila/track"
)

// ReumannWitkam implements the Reumann-Witkam
// algorithm that shifts a strip along the polyline
// and removes points that are within maxd meters.
func ReumannWitkam(maxd float64) Algorithm {
	return &reumannWitkam{maxd * maxd}
}

type reumannWitkam struct {
	dd float64 // meters squared
}

func (rw *reumannWitkam) run(dst, src track.Track) track.Track {
	if len(src) <= 2 {
		return append(dst, src...)
	}

	a := src[0]
	a3 := pt3(a)
	dst = append(dst, a)

	i := 1
	for i < len(src) {
		b := src[i]
		b3 := pt3(b)

		ut := float64(b.Time().Sub(a.Time())) // nanoseconds
		if ut < 1e3 {
			i++
			continue
		}

		// meters/nanosecond
		velocity := b3.Sub(a3).Muls(1 / ut)

		j := i + 1
		for ; j < len(src); j++ {
			c := src[j]
			c3 := pt3(c)

			dt := float64(c.Time().Sub(a.Time()))

			projected := a3.Add(velocity.Muls(dt))

			if dist3sq(c3, projected) > rw.dd {
				break
			}
		}

		a = src[j-1]
		a3 = pt3(a)
		dst = append(dst, a)
		i = j
	}
	return dst
}
