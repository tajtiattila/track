package track_test

import (
	"testing"

	"github.com/tajtiattila/track"
)

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
