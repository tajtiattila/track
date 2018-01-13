package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track/trackio"
)

type FilterCmd struct {
	start, end string

	date, span string

	json bool
	gpx  bool
}

func init() {
	cmdmain.Register("filter", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(FilterCmd)
		flags.StringVar(&c.end, "end", "", "remove point after time (RFC3339)")
		flags.StringVar(&c.start, "start", "", "remove points before time (RFC3339)")
		flags.StringVar(&c.date, "date", "", "filter on date (RFC3339)")
		flags.StringVar(&c.span, "span", "", "date filter span, one of year, month, day, hour (default inferred from -date)")
		flags.BoolVar(&c.json, "json", false, "print json output similar to google location history json")
		flags.BoolVar(&c.gpx, "gpx", false, "print gpx output (with no accuracy info)")
		return c
	})
}

func (*FilterCmd) Describe() string {
	return "Filter tracks."
}

func (*FilterCmd) ArgNames() string {
	return "[paths...]"
}

func (c *FilterCmd) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("need track path arguments")
	}

	if c.end == "" && c.start == "" && c.date == "" {
		return fmt.Errorf("need at least one of -start, -end or -date")
	}

	if c.end != "" && c.start != "" && c.date != "" {
		return fmt.Errorf("-date makes no sense with both -start and -end")
	}
	if c.span != "" && c.date == "" {
		return fmt.Errorf("-span needs -date")
	}

	var start, end time.Time
	if c.date != "" {
		t, prec, err := argTimePrec(c.date)
		if err != nil {
			return errors.Wrap(err, "invalid -date")
		}
		if c.span != "" {
			prec, err = strPrec(c.span)
			if err != nil {
				return errors.Wrap(err, "invalid -span")
			}
		}
		start, end = timeRange(t, prec)
	}
	if c.start != "" {
		t, err := argTime(c.start)
		if err != nil {
			return errors.Wrap(err, "invalid -start")
		}
		start = t
	}
	if c.end != "" {
		t, err := argTime(c.end)
		if err != nil {
			return errors.Wrap(err, "invalid -end")
		}
		end = t
	}

	for _, fn := range args {
		if err := c.filter(fn, start, end); err != nil {
			return err
		}
	}

	return nil
}

func (c *FilterCmd) filter(fn string, start, end time.Time) error {
	trk, err := load(fn)
	if err != nil {
		return err
	}

	si := trk.TimeIndex(start)
	if si > 0 {
		si--
	}
	ei := trk.TimeIndex(end)
	if ei < len(trk) {
		ei++
	}
	trk = trk[si:ei]

	if c.json {
		err = writeJSON(os.Stdout, trk)
	} else if c.gpx {
		err = writeGPX(os.Stdout, trk)
	} else {
		err = dumpTrack(os.Stdout, trk)
	}

	return err
}

func dumpTrack(w io.Writer, trk trackio.Track) error {
	xw := newErrWriter(w)
	for i, p := range trk {
		fmt.Fprintf(xw, "%4d %s %11.6f %11.6f", i,
			p.Time.UTC().Format("2006-01-02T15:04:05.000Z"),
			p.Lat, p.Long)
		if p.Acc < trackio.NoAccuracy {
			fmt.Fprintf(xw, "  %5.0f", p.Acc)
		} else {
			fmt.Fprintf(xw, "  %5s", "-")
		}
		fmt.Fprintln(xw)
	}
	return xw.Err()
}

func writeGPX(w io.Writer, trk trackio.Track) error {
	xw := newErrWriter(w)
	xw.WriteString(`<?xml version="1.0"?>
<gpx xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
`)
	for _, p := range trk {
		fmt.Fprintf(xw, `      <trkpt lat="%.6f" lon="%.6f">`, p.Lat, p.Long)
		fmt.Fprintf(xw, " <time>%s</time> </trkpt>\n", p.Time.UTC().Format(time.RFC3339))
	}

	xw.WriteString("    </trkseg>\n  </trk>\n</gpx>\n")
	return xw.Err()
}

func writeJSON(w io.Writer, trk trackio.Track) error {
	xw := newErrWriter(w)
	xw.WriteString(`{"locations" : [ `)
	for i, p := range trk {
		if i != 0 {
			xw.WriteString(", ")
		}

		var jp struct {
			Ts     string   `json:"timestampMs"`
			LatE7  float64  `json:"latitudeE7"`
			LongE7 float64  `json:"longitudeE7"`
			Acc    *float64 `json:"accuracy,omitempty"`
			Ele    *float64 `json:"altitude,omitempty"`
			VAcc   *float64 `json:"verticalAccuracy,omitempty"`
		}
		jp.Ts = fmt.Sprint(p.Time.UnixNano() / 1e6)
		jp.LatE7 = math.Floor(p.Lat*1e7 + 0.5)
		jp.LongE7 = math.Floor(p.Long*1e7 + 0.5)

		if p.Acc < trackio.NoAccuracy {
			jp.Acc = &p.Acc
		}
		if p.Ele.Valid {
			jp.Ele = &p.Ele.Float64
			if p.Ele.Acc < trackio.NoAccuracy {
				jp.VAcc = &p.Ele.Acc
			}
		}
		v, err := json.MarshalIndent(jp, " ", " ")
		if err != nil {
			panic(err)
		}
		xw.Write(v)
	}
	xw.WriteString(" ]\n}\n")
	return xw.Err()
}

type errWriter struct {
	w   io.Writer
	err error
}

func newErrWriter(w io.Writer) *errWriter {
	return &errWriter{w: w}
}

func (w *errWriter) Err() error { return w.err }

func (w *errWriter) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err = w.w.Write(p)
	w.err = err
	return n, err
}

func (w *errWriter) WriteString(s string) error {
	if w.err != nil {
		return w.err
	}
	_, err := w.Write([]byte(s))
	return err
}
