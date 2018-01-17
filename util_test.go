package track_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/tajtiattila/track"
)

type pointCmpFunc func(got, want track.Point) error

func pointCmpPackSame(got, want track.Point) error {
	dt := int64(got.Time().Sub(want.Time()) / time.Millisecond)
	if dt < 0 {
		dt = -dt
	}
	dlat := got.Lat() - want.Lat()
	dlong := got.Long() - want.Long()
	dd := math.Sqrt(dlat*dlat + dlong*dlong)
	if dt > 50 || dd > 2e-6 {
		return fmt.Errorf("got %v want %v", got, want)
	}
	return nil
}

func pointCmpSame(got, want track.Point) error {
	if got != want {
		return fmt.Errorf("got %v want %v", got, want)
	}
	return nil
}

func trackCmp(got, want track.Track, pcmp pointCmpFunc) error {
	if len(got) != len(want) {
		return fmt.Errorf("track lengths differ: got %d want %d", len(got), len(want))
	}

	for i, g := range got {
		w := want[i]
		if err := pcmp(w, g); err != nil {
			return fmt.Errorf("track point %d: %v", i, err)
		}
	}

	return nil
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

func humanSize(v uintptr) string {
	f := float64(v)
	suffix := []string{"", "K", "M", "G", "T", "P", "E"}
	for i, x := range suffix {
		if i != 0 && f < 10 {
			return fmt.Sprintf("%.1f%s", f, x)
		}
		if f < 1000 {
			return fmt.Sprintf("%.0f%s", f, x)
		}
		f /= 1024
	}
	return fmt.Sprint(v)
}
