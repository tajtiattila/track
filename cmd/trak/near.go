package main

import (
	"flag"
	"fmt"
	"math"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/geocode"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/geomath"
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
		trk.Merge(trackTrack(seg))
	}

	return c.find(gc, args[0], trk)
}

func (c *NearCmd) find(gc geocode.Geocoder, place string, trk track.Track) error {
	r, err := gc.Geocode(place)
	if err != nil {
		return err
	}

	if cli.verbose {
		fmt.Printf("Geocode result for %s is:\n%+v\n", place, r)
	}

	q3 := geomath.Pt3(r.Lat, r.Long)
	ne := geomath.Pt3(r.North, r.East)
	sw := geomath.Pt3(r.South, r.West)

	d2 := c.dist * c.dist
	if x := dist3sq(q3, ne); x > d2 {
		d2 = x
	}
	if x := dist3sq(q3, sw); x > d2 {
		d2 = x
	}

	if cli.verbose {
		fmt.Printf("d = %.2f\n", math.Sqrt(d2))
	}

	in := false
	for _, p := range trk {
		p3 := pt3(p)
		d := p3.Sub(q3)
		dd := d.Dot(d)
		nextin := dd <= d2
		if in != nextin {
			in = nextin
			if in {
				fmt.Println("enter>", p.Time())
			} else {
				fmt.Println("leave<", p.Time())
			}
		}
	}

	return nil
}

func dist3sq(a, b geomath.Point3) float64 {
	d := a.Sub(b)
	return d.Dot(d)
}
