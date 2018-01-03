package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tajtiattila/basedir"
	"github.com/tajtiattila/geocode"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/trackmath"
)

func load(fn string) (track.Track, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return track.Decode(f)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func argTime(s string) (time.Time, error) {
	fmts := []string{
		"2006-01-02 15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04Z",
		"2006-01-02 15:04",
		"2006-01-02 15",
		"2006-01-02",
		"2006-01",
		"2006",
	}
	for _, f := range fmts {
		t, err := time.ParseInLocation(f, s, time.UTC)
		if err != nil {
			f = strings.Replace(f, " ", "T", 1)
			t, err = time.ParseInLocation(f, s, time.UTC)
		}
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cant'parse time %q", s)
}

func newGeocoder() (geocode.Geocoder, error) {
	const varname = "GOOGLEMAPS_APIKEY"
	apikey := os.Getenv(varname)
	if apikey == "" {
		return nil, fmt.Errorf("%v env var missing", varname)
	}

	cachedir, err := basedir.Cache.EnsureDir("trak", 0777)
	if err != nil {
		return nil, err
	}

	qc, err := geocode.LevelDB(filepath.Join(cachedir, "geocode-cache.leveldb"))
	if err != nil {
		return nil, err
	}
	g := geocode.LatLong(
		geocode.OpenLocationCode(
			geocode.Cache(
				geocode.StdGoogle(apikey),
				qc)))
	return g, nil
}

func pt3(p track.Point) trackmath.Point3 {
	return trackmath.Pt3(p.Lat(), p.Long())
}
