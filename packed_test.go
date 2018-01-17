package track_test

import (
	"math"
	"testing"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/testutil"
)

func TestPackedUnpack(t *testing.T) {
	for _, f := range testutil.Files(t) {
		t.Log(f.Path())
		trk := f.Track(t)

		pk, _ := track.Pack(trk, 1, 1)

		xtrk := pk.Unpack(nil)

		if err := trackCmp(xtrk, trk, pointCmpPackSame); err != nil {
			t.Error(err)
		}
	}
}

func TestPackedLookup(t *testing.T) {
	for _, f := range testutil.Files(t) {
		t.Log(f.Path())
		trk := f.Track(t)

		t.Logf(" start: %s", trk.StartTime())
		t.Logf(" end: %s", trk.EndTime())
		pk, _ := track.Pack(trk, 1, 1)

		for _, tt := range testutil.TrackTimes(trk) {
			glat, glong := pk.At(tt)
			wlat, wlong := trk.At(tt)
			dlat := math.Abs(wlat - glat)
			dlong := math.Abs(wlong - glong)
			if dlat > 1.5e-6 || dlong > 1.5e-6 {
				t.Errorf("lookup: at %s got %.6f,%.6f want %.6f,%.6f",
					tt.Format("2006-01-02T15:04:05.000Z"), glat, glong, wlat, wlong)
			}
		}
	}
}
