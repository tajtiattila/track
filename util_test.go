package track_test

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/trackio"
)

var Files []testFile

func init() {
	tf := func(fn string) {
		Files = append(Files, testFile{filepath.Join("testdata", fn)})
	}

	tf("italy-slovenia-2017-07-29.json")
	tf("prague-2014-04-25.json")
}

type testFile struct {
	path string
}

func (tf testFile) String() string {
	return filepath.Base(tf.path)
}

func (tf testFile) open(t testing.TB) io.ReadCloser {
	r, err := os.Open(tf.path)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (tf testFile) track(t testing.TB) track.Track {
	f := tf.open(t)
	defer f.Close()

	var trk track.Track
	d := trackio.NewDecoder(f)
	for {
		pt, reset, err := d.Point()
		if err != nil {
			if err == io.EOF {
				return trk
			}
			t.Fatal(err)
		}
		if reset {
			trk = trk[:0]
		}
		trk = append(trk, track.Pt(
			pt.Time,
			pt.Lat,
			pt.Long,
		))
	}
	panic("unreachable")
}

func trackTestTimes(trk track.Track) []time.Time {
	var times []time.Time
	for i := -10; i < 10; i++ {
		dt := time.Duration(i) * time.Second
		times = append(times, trk.StartTime().Add(dt), trk.EndTime().Add(dt))
	}

	rnd := rand.New(rand.NewSource(trk.StartTime().Unix()))
	xt := int64(trk.EndTime().Sub(trk.StartTime()))
	if xt != 0 {
		for i := 0; i < 100; i++ {
			dt := time.Duration(rnd.Int63n(xt))
			times = append(times, trk.StartTime().Add(dt))
		}
	}
	return times
}

type pointCmpFunc func(got, want track.Point) error

func pointCmpSame(got, want track.Point) error {
	if got != want {
		return fmt.Errorf("gor %v want %v", got, want)
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
