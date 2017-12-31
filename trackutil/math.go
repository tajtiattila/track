package trackutil

import (
	"math"

	"github.com/tajtiattila/track"
)

const earthRadius = 6.371e6 // meters

const degToRad = math.Pi / 180

// point3 is a 3d coordinate of a point
// x [0] axis points at the equator at longitude 0
// y [1] axis points north
// z [1] axis points at the equator at longitude 90
type point3 [3]float64

func pt3(p track.Point) point3 {
	lat := p.Lat() * degToRad
	long := p.Long() * degToRad
	slat, clat := math.Sincos(lat)
	slong, clong := math.Sincos(long)
	return point3{
		earthRadius * clat * clong,
		earthRadius * slat,
		earthRadius * clat * slong,
	}
}

func (p point3) add(q point3) point3 {
	return point3{
		p[0] + q[0],
		p[1] + q[1],
		p[2] + q[2],
	}
}

func (p point3) sub(q point3) point3 {
	return point3{
		p[0] - q[0],
		p[1] - q[1],
		p[2] - q[2],
	}
}

func (p point3) muls(s float64) point3 {
	return point3{
		s * p[0],
		s * p[1],
		s * p[2],
	}
}

func mag3(p point3) float64 {
	return math.Sqrt(mag3sq(p))
}

func mag3sq(p point3) float64 {
	return math.Sqrt(p[0]*p[0] + p[1]*p[1] + p[2]*p[2])
}

func dist3(u, v point3) float64 {
	return math.Sqrt(dist3sq(u, v))
}

func dist3sq(u, v point3) float64 {
	x := u[0] - v[0]
	y := u[1] - v[1]
	z := u[2] - v[2]
	return x*x + y*y + z*z
}

func dot3(u, v point3) float64 {
	return u[0]*v[0] + u[1]*v[1] + u[2]*v[2]
}

func cross3(u, v point3) point3 {
	return point3{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	}
}

//      x            1 + cos x
// cos --- = ±sqrt( ----------- )
//      2                2
//
func angle3(u, v point3) float64 {
	// u · v == |u| |v| cos Θ
	// |u| == |v| == 0
	return math.Acos(dot3(u, v))
}
