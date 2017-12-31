package track_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/track"
)

func TestLookup(t *testing.T) {
	epoch := time.Date(2010, 6, 1, 10, 30, 0, 0, time.UTC)

	const u = 360 / 4e7 // ~1 meter

	var tests = []struct {
		what string
		trk  track.Track

		at          time.Time
		wlat, wlong float64
	}{
		{
			"pt[0]",
			timeTrkGen(epoch, -10, -10).trk(60, 0, 1),
			epoch,
			-10, -10,
		},
		{
			"pt[1]",
			timeTrkGen(epoch, -10, -10).trk(60, 0, 1),
			epoch.Add(time.Second),
			-10 + u, -10 + u,
		},
		{
			"midpt between pt[0] and pt[1]",
			timeTrkGen(epoch, -10, -10).trk(60, 0, 1),
			epoch.Add(time.Second / 2),
			-10 + u/2, -10 + u/2,
		},
		{
			"between pt[0] and pt[1]",
			timeTrkGen(epoch, -10, -10).trk(60, 0, 1),
			epoch.Add(time.Second / 4),
			-10 + u/4, -10 + u/4,
		},
		{
			"before start",
			timeTrkGen(epoch, -10, -10).trk(60, 0, 1),
			epoch.Add(-time.Second),
			-10, -10,
		},
		{
			"after end",
			timeTrkGen(epoch, -20, -20).trk(61, 0, 1),
			epoch.Add(time.Hour),
			-20 + 60*u, -20 + 60*u,
		},
		{
			"track over date line",
			track.Track{
				track.Pt(epoch, 0, 180-u),
				track.Pt(epoch.Add(2*time.Second), 2*u, -180+3*u),
			},
			epoch.Add(time.Second),
			u, -180 + u,
		},
		{
			"track over date line #2",
			track.Track{
				track.Pt(epoch, 2*u, -180+u),
				track.Pt(epoch.Add(2*time.Second), 0, 180-3*u),
			},
			epoch.Add(time.Second),
			u, 180 - u,
		},
	}

	for _, tt := range tests {

		// test for track errors
		lastt := epoch
		for _, pt := range tt.trk {
			if pt.Time().Before(lastt) {
				t.Fatal("unsorted test track")
			}
			lastt = pt.Time()
		}

		// test actual positions
		glat, glong := tt.trk.At(tt.at)

		// rounding errors can happen when passing over date line
		const eps = 1e-8

		dy := glat - tt.wlat
		dx := glong - tt.wlong
		if dx*dx+dy*dy > eps {
			t.Errorf("%s got (%.7f,%.7f) want (%.7f,%.7f)", tt.what,
				glat, glong,
				tt.wlat, tt.wlong)
		}
	}
}

func fsec(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

// timeTrackGenerator generates timed tracks
type timeTrackGenerator struct {
	epoch       time.Time
	lat, long   float64
	dlat, dlong float64
	i           uint
}

func timeTrkGen(epoch time.Time, lat, long float64) *timeTrackGenerator {
	const u = 360 / 4e7 // ~1 meter
	return &timeTrackGenerator{
		epoch: epoch,
		lat:   lat,
		long:  long,
		dlat:  u,
		dlong: u,
		i:     0,
	}
}

func (g *timeTrackGenerator) trk(n int, tofs, dt float64) track.Track {
	start := g.epoch.Add(fsec(tofs))

	trk := make(track.Track, n)
	for i := range trk {
		trk[i] = track.Pt(
			start.Add(time.Duration(i)*fsec(dt)),
			g.lat+float64(i)*g.dlat,
			g.long+float64(i)*g.dlong,
		)
	}

	if g.i == 0 {
		g.lat++
		g.dlat = -g.dlat
	} else {
		g.long++
		g.dlong = -g.dlong
	}
	g.i ^= 1

	return trk
}

var fakeTrackEpoch = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)

func fakeTrack(n int, tofs, dt, lat, long, dlat, dlong float64) track.Track {
	start := fakeTrackEpoch.Add(fsec(tofs))

	trk := make(track.Track, n)
	for i := range trk {
		trk[i] = track.Pt(
			start.Add(time.Duration(i)*fsec(dt)),
			lat+float64(i)*dlat,
			long+float64(i)*dlong,
		)
	}
	return trk
}
