package geomath_test

import (
	"math"
	"testing"

	"github.com/tajtiattila/track/geomath"
)

func TestConv(t *testing.T) {

	same := func(a, b float64) bool {
		const eps = 1e-7
		return math.Abs(a-b) < eps
	}

	for y := -75; y <= 75; y += 15 {
		for x := -165; x < 180; x += 40 {
			lat := float64(y)
			long := float64(x)
			p3 := geomath.Pt3(lat, long)
			if !same(p3.Mag(), geomath.EarthRadius) {
				t.Error("Point3 math invalid")
			}

			glat, glong := p3.LatLong()
			if !same(lat, glat) || !same(long, glong) {
				t.Errorf("%.2f,%.2f → %.2f,%.2f", lat, long, glat, glong)
			}
		}
	}
}
