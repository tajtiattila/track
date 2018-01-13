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

func (tf testFile) open(t *testing.T) io.ReadCloser {
	r, err := os.Open(tf.path)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (tf testFile) track(t *testing.T) track.Track {
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

const testpklen = 1024

func TestPackedUnpack(t *testing.T) {
	for _, f := range Files {
		t.Logf("testing %s\n", f.path)
		trk := f.track(t)

		pk, _ := track.Pack(trk, 1, 1, testpklen)

		xtrk := pk.Unpack(nil)

		if err := trackCmp(xtrk, trk, pointCmpSame); err != nil {
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
			if glat != wlat || glong != wlong {
				t.Errorf("lookup: at %s got %.6f,%.6f want %.6f,%.6f",
					tt, glat, glong, wlat, wlong)
			}
		}
	}
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
