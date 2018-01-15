package track_test

import (
	"math"
	"testing"

	"github.com/tajtiattila/track"
)

const testpklen = 1024

func TestPackedUnpack(t *testing.T) {
	for _, f := range Files {
		t.Logf("testing %s\n", f.path)
		trk := f.track(t)

		pk, _ := track.Pack(trk, 1, 1, testpklen)

		xtrk := pk.Unpack(nil)

		if err := trackCmp(xtrk, trk, pointCmpPackSame); err != nil {
			t.Error(err)
		}
	}
}

func TestPackedLookup(t *testing.T) {
	for _, f := range Files {
		t.Logf("testing %s\n", f.path)
		trk := f.track(t)

		t.Logf(" start: %s", trk.StartTime())
		t.Logf(" end: %s", trk.EndTime())
		pk, _ := track.Pack(trk, 1, 1, testpklen)

		for _, tt := range trackTestTimes(trk) {
			glat, glong := pk.At(tt)
			wlat, wlong := trk.At(tt)
			dlat := math.Abs(wlat - glat)
			dlong := math.Abs(wlong - glong)
			if dlat > 1.5e-6 || dlong > 1.5e-6 {
				t.Errorf("lookup: at %s got %.6f,%.6f want %.6f,%.6f",
					tt, glat, glong, wlat, wlong)
			}
		}
	}
}
