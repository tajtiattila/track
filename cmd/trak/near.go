package main

import (
	"flag"
	"fmt"
	"math"
	"time"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/geocode"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/internal/trackmath"
)

type NearCmd struct {
	dist float64
}

func init() {
	cmdmain.Register("near", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(NearCmd)
		flags.Float64Var(&c.dist, "d", 1000, "near distance in meters")
		return c
	})
}

func (*NearCmd) Describe() string {
	return "Find place in track(s)."
}

func (*NearCmd) ArgNames() string {
	return "[place] [paths...]"
}

func (c *NearCmd) Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("near needs a place and at least one track file")
	}

	gc, err := newGeocoder()
	if err != nil {
		return err
	}
	defer gc.Close()

	var trk track.Track
	for _, fn := range args[1:] {
		seg, err := load(fn)
		if err != nil {
			return err
		}
		trk.Merge(seg)
	}

	return c.find(gc, args[0], trk)
}

func (c *NearCmd) find(gc geocode.Geocoder, place string, trk track.Track) error {
	lat, long, err := gc.Geocode(place)
	if err != nil {
		return err
	}

	enter := c.dist * c.dist
	leave := enter * 4

	q3 := trackmath.Pt3(lat, long)

	var bestd float64
	var bestt time.Time
	show := func() {
		fmt.Println(bestt, math.Sqrt(bestd))
	}
	entered := false
	for _, p := range trk {
		p3 := pt3(p)
		d := p3.Sub(q3)
		dd := d.Dot(d)
		if dd < enter {
			entered = true
		}
		if entered && dd > leave {
			entered = false
			show()
			bestd, bestt = 0, time.Time{}
		}
		if bestt.IsZero() || dd < bestd {
			bestd, bestt = dd, p.Time()
		}
	}
	show()

	return nil
}
