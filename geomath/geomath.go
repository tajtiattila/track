// Package geomath provides geographical math utilities.
package geomath

import (
	"math"
)

const EarthRadius = 6.371e6 // meters

const degToRad = math.Pi / 180
const radToDeg = 180 / math.Pi

// Point3 is a 3d coordinate of a point
// x [0] axis points at the equator at longitude 0
// y [1] axis points north
// z [1] axis points at the equator at longitude 90
type Point3 [3]float64

func Pt3(latDeg, longDeg float64) Point3 {
	lat := latDeg * degToRad
	long := longDeg * degToRad
	slat, clat := math.Sincos(lat)
	slong, clong := math.Sincos(long)
	return Point3{
		EarthRadius * clat * clong,
		EarthRadius * slat,
		EarthRadius * clat * slong,
	}
}

func (p Point3) LatLong() (lat, long float64) {
	lat = math.Asin(p[1]/p.Mag()) * radToDeg
	long = math.Atan2(p[2], p[0]) * radToDeg
	return
}

func (p Point3) Add(q Point3) Point3 {
	return Point3{
		p[0] + q[0],
		p[1] + q[1],
		p[2] + q[2],
	}
}

func (p Point3) Sub(q Point3) Point3 {
	return Point3{
		p[0] - q[0],
		p[1] - q[1],
		p[2] - q[2],
	}
}

func (p Point3) Muls(s float64) Point3 {
	return Point3{
		s * p[0],
		s * p[1],
		s * p[2],
	}
}

func (p Point3) Mag() float64 {
	return math.Sqrt(p.Dot(p))
}

func (u Point3) Dot(v Point3) float64 {
	return u[0]*v[0] + u[1]*v[1] + u[2]*v[2]
}

func (u Point3) Cross(v Point3) Point3 {
	return Point3{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	}
}

//      x            1 + cos x
// cos --- = ±sqrt( ----------- )
//      2                2
//
func angle3(u, v Point3) float64 {
	// u · v == |u| |v| cos Θ
	// |u| == |v| == 0
	return math.Acos(u.Dot(v))
}
