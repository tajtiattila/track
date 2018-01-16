package testutil

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/tajtiattila/basedir"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/trackio"
)

var files []File

func init() {
	for _, p := range filepath.SplitList(os.Getenv("TESTTRACK")) {
		files = append(files, File{p})
	}

	dir, err := basedir.Gopath.Dir("src/github.com/tajtiattila/track")
	if err == nil {
		tf := func(fn string) {
			files = append(files, File{filepath.Join(dir, "testdata", fn)})
		}

		tf("italy-slovenia-2017-07-29.json")
		tf("prague-2014-04-25.json")
	}

}

func Files(t testing.TB) []File {
	if len(files) == 0 {
		t.Fatal("testutil: no test files")
	}
	return files
}

type File struct {
	path string
}

func (tf File) Path() string {
	return tf.path
}

func (tf File) Open(t testing.TB) io.ReadCloser {
	r, err := os.Open(tf.path)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (tf File) Track(t testing.TB) track.Track {
	trk := tf.load(t)
	trk.Sort()
	return trk
}

func (tf File) load(t testing.TB) track.Track {
	f := tf.Open(t)
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
