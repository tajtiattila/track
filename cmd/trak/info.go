package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/trackio"
	"github.com/tajtiattila/track/tracksimpl"
)

type InfoCmd struct {
	freq  bool
	round time.Duration
	maxd  float64
}

func init() {
	cmdmain.Register("info", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(InfoCmd)
		flags.BoolVar(&c.freq, "freq", false, "frequency analysis")
		flags.DurationVar(&c.round, "round", 0, "round times (0: off)")
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
		i.distAnalyze(trk)
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

func (i *InfoCmd) distAnalyze(trk0 track.Track) {
	trk := tracksimpl.RoundTime{i.round}.Run(nil, trk0)
	fmt.Printf(" times rounded to %v: %d\n", i.round, len(trk))

	if i.maxd <= 0 {
		return
	}

	x := make(track.Track, len(trk))

	showResultTime(trk0, " radial distance", func() track.Track {
		return tracksimpl.RadialDistance{D: i.maxd}.Run(x[:0], trk)
	})

	showResultTime(trk0, " reumann-witkam", func() track.Track {
		return tracksimpl.ShiftSegment{D: i.maxd}.Run(x[:0], trk)
	})

	showResultTime(trk0, " simplified", func() track.Track {
		return tracksimpl.Run(x[:0], trk, tracksimpl.ShiftSegment{D: i.maxd}, tracksimpl.RadialDistance{D: i.maxd})
	})

	showResultTime(trk, " end-point fit", func() track.Track {
		return tracksimpl.EndPointFit{D: i.maxd}.Run(x[:0], trk)
	})

}

func showResult(trk track.Track, pfx string, nkeep int) {
	ndrop := len(trk) - nkeep
	perc := float64(ndrop) / float64(len(trk)) * 100
	fmt.Printf("%s: %5.2f%% (%d points) dropped\n", pfx, perc, ndrop)
}

func showResultTime(trk track.Track, pfx string, f func() track.Track) {
	t0 := time.Now()
	x := f()
	dt := time.Now().Sub(t0)
	ndrop := len(trk) - len(x)
	perc := float64(ndrop) / float64(len(trk)) * 100
	fmt.Printf("%s: %5.2f%% (%d points) dropped; took %s\n", pfx, perc, ndrop, dt)
}
