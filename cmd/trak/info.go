package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/exp/tracksimpl"
)

type InfoCmd struct {
	freq bool
	maxd float64
}

func init() {
	cmdmain.Register("info", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(InfoCmd)
		flags.BoolVar(&c.freq, "freq", false, "frequency analysis")
		flags.Float64Var(&c.maxd, "maxd", 0, "run track simplification test with maxd in meters (0: off)")
		return c
	})
}

func (*InfoCmd) Describe() string {
	return "Show general track file infos."
}

func (*InfoCmd) ArgNames() string {
	return "[paths...]"
}

func (i *InfoCmd) Run(args []string) error {
	for _, fn := range args {
		i.trackInfo(fn)
	}
	return nil
}

func (i *InfoCmd) trackInfo(fn string) {
	f, err := os.Open(fn)
	check(err)
	defer f.Close()

	trk, err := track.Decode(f)
	check(err)

	fmt.Printf("%s:\n %d points\n", fn, len(trk))
	if len(trk) != 0 {
		fmt.Printf(" start: %s\n", trk[0].Time())
		fmt.Printf(" end: %s\n", trk[len(trk)-1].Time())
		i.stillAnalyze(trk)
		if i.freq {
			freqAnalyze(trk)
		}
	}
}

func freqAnalyze(trk track.Track) {
	if len(trk) == 0 {
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

func (i *InfoCmd) stillAnalyze(trk track.Track) {
	if i.maxd <= 0 {
		return
	}

	x := tracksimpl.Run(nil, trk, tracksimpl.StillFilter(i.maxd))
	showResult(trk, " still filter", len(x))

	x = tracksimpl.Run(nil, trk, tracksimpl.ReumannWitkam(i.maxd))
	showResult(trk, " reumann-witkam", len(x))

	x = tracksimpl.Run(nil, trk, tracksimpl.ReumannWitkam(i.maxd), tracksimpl.StillFilter(i.maxd))
	showResult(trk, " simplified", len(x))
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
