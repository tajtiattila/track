package track_test

import (
	"testing"

	"github.com/tajtiattila/track"
)

func pointEqual(t *testing.T, got, want track.Point) {
	if !got.T.Equal(want.T) {
		t.Fatalf("got time %v, want %v", got.T, want.T)
	}

	if got.Lat != want.Lat {
		t.Fatalf("got latitude %v, want %v", got.Lat, want.Lat)
	}

	if got.Long != want.Long {
		t.Fatalf("got latitude %v, want %v", got.Long, want.Long)
	}
}
