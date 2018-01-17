package tracksimpl

import (
	"github.com/tajtiattila/track"
)

// ShiftSegment implements the Reumann-Witkam
// algorithm that shifts a strip along the polyline
// and removes points that are within D meters.
//
// It has O(n) complexity.
type ShiftSegment struct {
	D float64 // maximum distance in meters

	Strict bool // emit only points from the source
}

func (ss ShiftSegment) Run(dst, src track.Track) track.Track {
	if len(src) <= 2 {
		return append(dst, src...)
	}

	d := ss.D
	if ss.Strict {
		// allowed distance must be halved
		// to fulfull package guarantee
		d /= 2
	} else {
		// account for possible 3d to lat/long conversion error
		d -= 0.01
	}

	if d <= 0 {
		return append(dst, src...)
	}

	dd := d * d

	a := src[0]
	a3 := pt3(a)
	dst = append(dst, a)

	i := 1
	for i < len(src) {
		b := src[i]
		b3 := pt3(b)
		bt := b.Time()

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
			ct := c.Time()

			dt := float64(ct.Sub(a.Time()))

			projected := a3.Add(velocity.Muls(dt))

			if dist3sq(c3, projected) > dd {
				break
			}

			b3 = projected
			bt = ct
		}

		a3 = b3

		if ss.Strict {
			a = src[j-1]
		} else {
			lat, long := a3.LatLong()
			a = track.Pt(bt, lat, long)
		}

		dst = append(dst, a)
		i = j
	}
	return dst
}
