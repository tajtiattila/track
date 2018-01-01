package track_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/track"
)

func TestMerge(t *testing.T) {
	mt := func(what string, trk, seg track.Track, sn, en int) {

		var want track.Track
		want = append(want, trk[:sn]...)
		want = append(want, seg...)
		want = append(want, trk[len(trk)-en:]...)

		trk.Merge(seg)

		if len(want) != len(trk) {
			t.Fatalf("%s: got len %d, want %d", what, len(trk), len(want))
			return
		}

		for i := range trk {
			pointEqual(t, trk[i], want[i])
		}
	}

	epoch := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)

	gen := timeTrkGen(epoch, 46, 17)

	mt("trk before seg",
		gen.trk(5, 0, 1.0),
		gen.trk(5, 60, 1.0),
		5, 0)

	mt("trk after seg",
		gen.trk(5, 60, 1.0),
		gen.trk(5, 0, 1.0),
		0, 5)

	mt("seg overwrites trk",
		gen.trk(5, 10, 1.0),
		gen.trk(60, 0, 1.0),
		0, 0)

	mt("seg within trk",
		gen.trk(60, 0, 1.0),
		gen.trk(10, 10, 1.0),
		10, 40)

	mt("long seg within trk",
		gen.trk(60, 0, 1.0),
		gen.trk(10, 10, 0.5),
		10, 45)

	mt("short seg within trk",
		gen.trk(60, 0, 1.0),
		gen.trk(10, 10, 2.0),
		10, 31)

	mt("seg overlaps with start of trk",
		gen.trk(60, 30, 1.0),
		gen.trk(60, 0, 1.0),
		0, 30)

	mt("seg overlaps with end of trk",
		gen.trk(60, 0, 1.0),
		gen.trk(60, 30, 1.0),
		30, 0)

	mt("seg overlaps with start of trk #2",
		gen.trk(60, 30, 1.0),
		gen.trk(60, 0.5, 1.0),
		0, 30)

	mt("seg overlaps with end of trk #2",
		gen.trk(60, 0.5, 1.0),
		gen.trk(60, 30, 1.0),
		30, 0)
}
