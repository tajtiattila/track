package tracksimpl_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/tracksimpl"
)

type trkgen struct {
	start, t, lastt time.Time
	lat, long       float64 // origin

	trk track.Track
}

func newtrkgen(lat, long float64) *trkgen {
	t, err := time.ParseInLocation(time.RFC3339, "2018-01-01T00:00:00Z", time.UTC)
	if err != nil {
		panic(err)
	}
	return &trkgen{
		start: t,
		t:     t,
		lat:   lat,
		long:  long,
	}
}

func (g *trkgen) pt(v ...float64) {
	for i := 0; i < len(v); i += 2 {
		lat, long := v[i], v[i+1]
		g.lastt = g.t
		g.t = g.t.Add(time.Second)
		g.trk = append(g.trk, track.Pt(g.lastt, lat, long))
	}
}

func (g *trkgen) m(xy ...float64) {
	for i := 0; i < len(xy); i += 2 {
		x, y := xy[i], xy[i+1]
		const m = 360 / 40e6 // meters to degrees
		lat := g.lat + y*m
		long := g.long + x*m
		g.pt(lat, long)
	}
}

func TestRadialDistance(t *testing.T) {
	var origins = []struct {
		lat, long float64
	}{
		{0, 0},
		//{0, 180},
	}

	for _, o := range origins {
		g := newtrkgen(o.lat, o.long)

		g.m(0, -100)

		// pts around origin
		g.m(
			0, 0,
			1, 1,
			1, -1,
			-1, -1,
			-1, 1,
			0, 0,
		)

		g.m(0, 100)

		dst := tracksimpl.Run(nil, g.trk, tracksimpl.RadialDistance{5})
		if len(dst) != 4 {
			t.Errorf("got length %v, want 4", len(dst))
		} else {
			pointEqual(t, dst[1], track.Pt(g.start.Add(time.Second), o.lat, o.long))
			pointEqual(t, dst[2], track.Pt(g.start.Add(6*time.Second), o.lat, o.long))
		}
	}
}

func pointEqual(t *testing.T, got, want track.Point) {
	if !got.Time().Equal(want.Time()) {
		t.Fatalf("got time %v, want %v", got.Time(), want.Time())
	}

	if got.Lat() != want.Lat() {
		t.Fatalf("got latitude %v, want %v", got.Lat(), want.Lat())
	}

	if got.Long() != want.Long() {
		t.Fatalf("got latitude %v, want %v", got.Long(), want.Long())
	}
}
