package tracksimpl_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/testutil"
	"github.com/tajtiattila/track/internal/trackmath"
	"github.com/tajtiattila/track/tracksimpl"
)

var algos []algoTest
var maxAlgoNameLen int

type algoTest struct {
	name string

	dist float64

	algo tracksimpl.Algorithm
}

func init() {
	algo := func(name string, fn func(d float64) tracksimpl.Algorithm) {
		dists := []float64{1, 2, 5, 10}
		for _, d := range dists {
			n := fmt.Sprintf("%s(%.1f)", name, d)
			if len(n) > maxAlgoNameLen {
				maxAlgoNameLen = len(n)
			}
			algos = append(algos, algoTest{
				name: n,
				dist: d,
				algo: fn(d),
			})
		}
	}

	algo("EndPointFit", func(d float64) tracksimpl.Algorithm {
		return tracksimpl.EndPointFit{D: d}
	})
	algo("RadialDistance", func(d float64) tracksimpl.Algorithm {
		return tracksimpl.RadialDistance{D: d}
	})
	algo("ShiftSegment", func(d float64) tracksimpl.Algorithm {
		return tracksimpl.ShiftSegment{D: d}
	})
	algo("ShiftSegment.Strict", func(d float64) tracksimpl.Algorithm {
		return tracksimpl.ShiftSegment{D: d, Strict: true}
	})
}

func TestAlgos(t *testing.T) {
	var n int                  // total track points
	an := make(map[string]int) // algo name to result points
	for _, f := range testutil.Files(t) {
		trk := f.Track(t)

		n += len(trk)

		t.Log(f.Path())
		t.Logf(" start: %s", ts(trk.StartTime()))
		t.Logf(" end: %s", ts(trk.EndTime()))
		t.Logf(" %d points\n", len(trk))

		for _, a := range algos {
			x := a.algo.Run(nil, trk)

			an[a.name] += len(x)
			compareResult(t, trk, x, a.name, a.dist)
		}
	}

	for _, a := range algos {
		frac := float64(an[a.name]) / float64(n)
		t.Logf(" %*s %.2f%% %d/%d\n", -maxAlgoNameLen, a.name, 100*frac, an[a.name], n)
	}
}

func BenchmarkAlgos(b *testing.B) {
	for _, f := range testutil.Files(b) {
		trk := f.Track(b)

		x := make(track.Track, 0, len(trk))

		for _, a := range algos {
			n := filepath.Base(f.Path()) + "/" + a.name
			b.Run(n, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					x = a.algo.Run(x[:0], trk)
				}
			})
		}
	}
}

func compareResult(t testing.TB, base, result track.Track, name string, dist float64) {
	times := testutil.TrackTimes(base)
	var dmax, dsum float64
	var n int
	for tt := range times {
		wlat, wlong := base.At(tt)
		want := trackmath.Pt3(wlat, wlong)
		glat, glong := result.At(tt)
		got := trackmath.Pt3(glat, glong)
		d := want.Sub(got).Mag()
		if d > dist {
			t.Errorf("%s: at %s got %.6f,%.6f want %.6f,%.6f (dist %f > %f)",
				name, ts(tt), glat, glong, wlat, wlong, d, dist)
		}

		dsum += d
		if d > dmax {
			dmax = d
		}
		n++
	}
	davg := dsum / float64(n)
	t.Logf("%s distance error: avg %.3f max %.3f", name, davg, dmax)
}
