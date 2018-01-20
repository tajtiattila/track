package tracksimpl_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/track"
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

func TestEndPointFitWorstCase(t *testing.T) {
	const dist = 5
	trk := entPointFitWorstCase(10*1024, dist)
	full := tracksimpl.EndPointFit{
		D:    dist,
		Full: true,
	}.Run(nil, trk)
	if len(full) != len(trk) {
		t.Errorf("full: got length %d, want %d", len(full), len(trk))
	}

	a := tracksimpl.EndPointFit{
		D: dist,
	}.Run(nil, trk)
	if len(a) != len(trk) {
		t.Errorf("adaptive: got length %d, want %d", len(a), len(trk))
	}
}

func BenchmarkEndPointFitWorstCase(b *testing.B) {
	const dist = 5
	trk := entPointFitWorstCase(10*1024, dist)
	x := make(track.Track, len(trk))
	b.Run("full", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x = tracksimpl.EndPointFit{
				D:    dist,
				Full: true,
			}.Run(x[:0], trk)
		}
	})
	b.Log(len(x))

	b.Run("adaptive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x = tracksimpl.EndPointFit{
				D: dist,
			}.Run(x[:0], trk)
		}
	})
	b.Log(len(x))
}
