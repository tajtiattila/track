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
	"github.com/tajtiattila/track/geomath"
	"github.com/tajtiattila/track/trackio"
)

func load(fn string) (trackio.Track, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := trackio.NewDecoder(f)
	if cli.inacc {
		d.Accuracy = trackio.NoAccuracy
	}
	return d.Track()
}

func loadRaw(fn string) (trackio.Track, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := trackio.NewDecoder(f)
	d.Accuracy = trackio.NoAccuracy
	return d.Track()
}

func trackTrack(t0 trackio.Track) track.Track {
	trk := make(track.Track, len(t0))
	for i, p := range t0 {
		trk[i] = track.Pt(p.Time, p.Lat, p.Long)
	}
	return trk
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func argTimePrec(s string) (time.Time, int, error) {
	fmts := []struct {
		layout string
		prec   int
	}{
		{"2006-01-02 15:04:05Z", 6},
		{"2006-01-02 15:04:05", 6},
		{"2006-01-02 15:04Z", 5},
		{"2006-01-02 15:04", 5},
		{"2006-01-02 15", 4},
		{"2006-01-02", 3},
		{"2006-01", 2},
		{"2006", 1},
	}
	for _, f := range fmts {
		t, err := time.Parse(f.layout, s)
		if err != nil {
			x := strings.Replace(f.layout, " ", "T", 1)
			t, err = time.Parse(x, s)
		}
		if err == nil {
			return t.UTC(), f.prec, nil
		}
	}
	return time.Time{}, 0, fmt.Errorf("cant'parse time %q", s)
}

func argTime(s string) (time.Time, error) {
	t, _, err := argTimePrec(s)
	return t, err
}

func strPrec(s string) (int, error) {
	switch s {
	case "year":
		return 1, nil
	case "month":
		return 2, nil
	case "day":
		return 3, nil
	case "hour":
		return 4, nil
	}
	return 0, fmt.Errorf("unknown precision string %q", s)
}

func timeRange(t time.Time, prec int) (start, end time.Time) {
	t = t.Local()
	switch prec {
	case 1: // year
		start = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
		end = start.AddDate(1, 0, 0)
	case 2: // month
		y, m, _ := t.Date()
		start = time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
		end = start.AddDate(0, 1, 0)
	case 3: // day
		y, m, d := t.Date()
		start = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		end = start.AddDate(0, 0, 1)
	default:
		start = t.Round(time.Hour)
		end = t.Add(time.Hour)
	}
	return
}

func newGeocoder() (geocode.Geocoder, error) {
	const varname = "GOOGLEMAPS_APIKEY"
	apikey := os.Getenv(varname)

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

func pt3(p track.Point) geomath.Point3 {
	return geomath.Pt3(p.Lat(), p.Long())
}

func trackTimeRange(trk track.Track, start, end time.Time) track.Track {
	si := trk.TimeIndex(start)
	if si > 0 {
		si--
	}
	ei := trk.TimeIndex(end)
	if ei < len(trk) {
		ei++
	}
	return trk[si:ei]
}
