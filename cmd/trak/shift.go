package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track"
	"github.com/tajtiattila/track/geomath"
)

type ShiftCmd struct {
	dist float64
	dt   time.Duration
}

func init() {
	cmdmain.Register("shift", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(ShiftCmd)
		flags.Float64Var(&c.dist, "d", 1000, "near distance in meters")
		flags.DurationVar(&c.dt, "dt", 72*time.Hour, "delta to time argument to check track against")
		return c
	})
}

func (*ShiftCmd) Describe() string {
	return "Calculate time shift for place and time in track."
}

func (*ShiftCmd) ArgNames() string {
	return "[place] [time] [paths...]"
}

func (c *ShiftCmd) Run(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("need a place and time, and at least one track file")
	}

	gc, err := newGeocoder()
	if err != nil {
		return err
	}
	defer gc.Close()

	r, err := gc.Geocode(args[0])
	if err != nil {
		return err
	}

	t0, err := argTime(args[1])
	if err != nil {
		return err
	}

	var trk track.Track
	for _, fn := range args[2:] {
		seg, err := load(fn)
		if err != nil {
			return err
		}
		trk.Merge(trackTrack(seg))
	}

	trk = trackTimeRange(trk, t0.Add(-c.dt), t0.Add(c.dt))

	return c.shift(r.Lat, r.Long, t0, trk)
}

func (c *ShiftCmd) shift(lat, long float64, orgt time.Time, trk track.Track) error {
	if len(trk) == 0 {
		return fmt.Errorf("empty track segment")
	}

	p3 := geomath.Pt3(lat, long)

	a := trk[0]
	a3 := pt3(a)

	tsf := func(t time.Time) string {
		return t.Local().Format("2006-01-02T15:04:05.0")
	}
	show := func(t time.Time, d float64) {
		fmt.Println(tsf(t), t.Sub(orgt), d)
	}

	var bestd float64
	var bestt time.Time

	for _, b := range trk[1:] {
		b3 := pt3(b)

		fmt.Println(" ", tsf(a.Time()))

		d, rel := segpt(p3, a3, b3)
		if true { //rel >= 0 && rel < 1 {

			at, bt := a.Time(), b.Time()
			t := at.Add(time.Duration(float64(bt.Sub(at)) * rel))
			show(t, d)

			if bestt.IsZero() || d < bestd {
				bestd = d
				bestt = t
			}
		}

		a, a3 = b, b3
	}

	show(bestt, bestd)

	return nil
}

func segpt(p, p0, p1 geomath.Point3) (dist, rel float64) {
	v := p1.Sub(p0)
	w := p.Sub(p0)

	c1 := w.Dot(v)
	if c1 <= 0 { // before p0
		return w.Mag(), 0
	}
	c2 := v.Dot(v)
	if c2 <= c1 { // after p1
		return p.Sub(p1).Mag(), 1
	}

	m := c1 / c2
	pb := p0.Add(v.Muls(m))
	return p.Sub(pb).Mag(), m
}
