package main

import (
	"flag"
	"fmt"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/trackio"
	"github.com/tajtiattila/track/tracksimpl"
)

type RelCmd struct {
}

func init() {
	cmdmain.Register("rel", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(RelCmd)
		return c
	})
}

func (*RelCmd) Describe() string {
	return "Test relative packing on tracks."
}

func (*RelCmd) ArgNames() string {
	return "[paths...]"
}

func (c *RelCmd) Run(args []string) error {
	for _, fn := range args {
		trk, err := load(fn)
		if err != nil {
			return err
		}
		c.run(fn, trk)
	}
	return nil
}

func (c *RelCmd) run(fn string, t0 trackio.Track) {
	trk := make(track.Track, len(t0))
	for i, p := range t0 {
		trk[i] = track.Pt(p.Time, p.Lat, p.Long)
	}

	fmt.Printf("%s: %d points\n", fn, len(trk))

	track.RelPack(trk)

	trk = tracksimpl.Run(nil, trk, tracksimpl.ShiftSegment{D: 2})

	fmt.Printf("  reumann-witkam(2) %d points\n", len(trk))

	track.RelPack(trk)
}
