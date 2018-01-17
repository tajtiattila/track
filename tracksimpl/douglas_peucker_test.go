package tracksimpl_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/testutil"
	"github.com/tajtiattila/track/tracksimpl"
)

func entPointFitWorstCase(n int, d float64) track.Track {

	t := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)

	m := float64(360) / 40e6 // ~ 1 meter

	dlong := m
	dlat := float64(m) / 10

	var trk track.Track
	for i := 0; i < n; i++ {
		lat := d + float64(i+1)*dlat
		if i%2 == 1 {
			// zigzag
			lat = -lat
		}
		long := float64(i) * dlong

		trk = append(trk, track.Pt(t, lat, long))

		t = t.Add(time.Second)
	}

	return trk
}

func TestEndPointFitWindow(t *testing.T) {
	for _, f := range testutil.Files(t) {
		trk := f.Track(t)

		n := len(trk)

		const dist = 5
		for w := n - 2; w <= n+2; w++ {
			x := tracksimpl.EndPointFit{
				D:      dist,
				Window: w,
			}.Run(nil, trk)

			compareResult(t, trk, x, fmt.Sprint("Window=", w), dist)
		}
	}
}

func BenchmarkEndPointFitWorstCase(b *testing.B) {
	const dist = 5
	const win = 1024
	trk := entPointFitWorstCase(10*win, dist)
	x := make(track.Track, len(trk))
	b.Run("nowin", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x = tracksimpl.EndPointFit{
				D: dist,
			}.Run(x[:0], trk)
		}
	})
	b.Log(len(x))
	for _, w := range []int{64, 256, 1024} {
		b.Run(fmt.Sprintf("win%d", w), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				x = tracksimpl.EndPointFit{
					D:      dist,
					Window: w,
				}.Run(x[:0], trk)
			}
		})
		b.Log(w, ": ", len(x))
	}
}
