package tracksimpl_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/testutil"
	"github.com/tajtiattila/track/internal/trackmath"
	"github.com/tajtiattila/track/tracksimpl"
)

func TestShiftSegment(t *testing.T) {
	dists := []float64{1, 2, 5, 10}
	for _, f := range testutil.Files(t) {
		trk := f.Track(t)

		t.Log(f.Path())
		t.Logf(" start: %s", ts(trk.StartTime()))
		t.Logf(" end: %s", ts(trk.EndTime()))

		times := testutil.TrackTimes(trk)

		var dst track.Track
		for _, sd := range dists {
			dst = tracksimpl.Run(dst[:0], trk, tracksimpl.ShiftSegment{D: sd})

			for tt := range times {
				wlat, wlong := trk.At(tt)
				want := trackmath.Pt3(wlat, wlong)
				glat, glong := dst.At(tt)
				got := trackmath.Pt3(glat, glong)
				d := want.Sub(got).Mag()
				if d > sd {
					t.Errorf("at %s got %.6f,%.6f want %.6f,%.6f (dist %f > %f)",
						ts(tt), glat, glong, wlat, wlong, d, sd)
				}
			}
		}
	}
}

func ts(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000Z")
}
