package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/trackutil"
)

var freq bool
var stillDist float64

func main() {
	flag.BoolVar(&freq, "freq", false, "frequency analysis")
	flag.Float64Var(&stillDist, "still", 0, "still analysis distance in meters (0: off)")
	flag.Parse()

	for _, a := range flag.Args() {
		trackInfo(a)
	}
}

func trackInfo(fn string) {
	f, err := os.Open(fn)
	check(err)
	defer f.Close()

	trk, err := track.Decode(f)
	check(err)

	fmt.Printf("%s:\n %d points\n", fn, len(trk))
	if len(trk) != 0 {
		fmt.Printf(" start: %s\n", trk[0].Time())
		fmt.Printf(" end: %s\n", trk[len(trk)-1].Time())
		stillAnalyze(trk)
		freqAnalyze(trk)
	}
}

func freqAnalyze(trk track.Track) {
	if !freq || len(trk) == 0 {
		return
	}

	freqs := []time.Duration{
		time.Second,
		15 * time.Second,
		30 * time.Second,
		time.Minute,
		2 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		time.Hour,
	}

	fmt.Println(" Frequency analysis:")
	for _, freq := range freqs {
		last, n := trk[0].Time().Truncate(freq), 1
		for _, p := range trk {
			pt := p.Time().Truncate(freq)
			if !pt.Equal(last) {
				last, n = pt, n+1
			}
		}
		showResult(trk, fmt.Sprintf("  %10s", freq), n)
	}
}

func stillAnalyze(trk track.Track) {
	t := trackutil.StillFilter(nil, trk, stillDist)
	showResult(trk, " near still", len(t))
}

func showResult(trk track.Track, pfx string, nkeep int) {
	ndrop := len(trk) - nkeep
	perc := float64(ndrop) / float64(len(trk)) * 100
	fmt.Printf("%s: %5.2f%% (%d points) dropped\n", pfx, perc, ndrop)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
