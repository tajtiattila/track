package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/exp/tracksimpl"
	"github.com/tajtiattila/track/trackio"
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
	if len(args) == 0 {
		return fmt.Errorf("need track path arguments")
	}
	for _, fn := range args {
		i.trackInfo(fn)
	}
	return nil
}

func (i *InfoCmd) trackInfo(fn string) {
	f, err := os.Open(fn)
	check(err)
	defer f.Close()

	d := trackio.NewDecoder(f)
	if cli.inacc {
		d.Accuracy = trackio.NoAccuracy
	}

	var trk track.Track
	for {
		pt, reset, err := d.Point()
		if err == io.EOF {
			break
		}
		check(err)
		if reset {
			trk = trk[:0]
		}
		trk = append(trk, track.Pt(pt.Time, pt.Lat, pt.Long))
	}
	trk.Sort()

	fmt.Printf("%s:\n %d points\n", fn, len(trk))
	if len(trk) != 0 {
		fmt.Printf(" start: %s\n", trk[0].Time())
		fmt.Printf(" end: %s\n", trk[len(trk)-1].Time())
		memSize(trk, " ")
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
	memSize(x, "  ")

	x = tracksimpl.Run(nil, trk, tracksimpl.ReumannWitkam(i.maxd))
	showResult(trk, " reumann-witkam", len(x))
	memSize(x, "  ")

	x = tracksimpl.Run(nil, trk, tracksimpl.ReumannWitkam(i.maxd), tracksimpl.StillFilter(i.maxd))
	showResult(trk, " simplified", len(x))
	memSize(x, "  ")
}

func memSize(trk track.Track, pfx string) {
	fmt.Printf("%smemsize: %.2fM\n", pfx, float64(len(trk)*16)/1e6)
	_, s := track.Pack(trk, 100, 10, 1024)
	fmt.Printf("%smemsize (packed): %.2fM\n", pfx, float64(s.MemSize)/1e6)
}

func showResult(trk track.Track, pfx string, nkeep int) {
	ndrop := len(trk) - nkeep
	perc := float64(ndrop) / float64(len(trk)) * 100
	fmt.Printf("%s: %5.2f%% (%d points) dropped\n", pfx, perc, ndrop)
}
