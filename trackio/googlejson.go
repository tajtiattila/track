package trackio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
)

func init() {
	RegisterFormat("googlejson", isGoogleJSON, newGoogleJSON)
}

func isGoogleJSON(p []byte) bool {
	j := json.NewDecoder(bytes.NewReader(p))
	return readGoogleJSONPrefix(j) == nil
}

func readGoogleJSONPrefix(j *json.Decoder) error {
	return readTokens(j,
		json.Delim('{'),
		"locations",
		json.Delim('['))
}

/* Google location history JSON format:

{"locations": [ {
    "timestampMs" : "1513838344568",
    "latitudeE7" : 461971931,
    "longitudeE7" : 182739971,
    "accuracy" : 20,
    "altitude" : 342,
    "verticalAccuracy" : 2,
    "activity" : [ ... ],
  }, { "timestampMs": ... }
]}

*/

func newGoogleJSON(r io.Reader) (PointReader, error) {
	j := json.NewDecoder(r)
	if err := readGoogleJSONPrefix(j); err != nil {
		panic("impossible")
	}

	return &googleJSON{j: j}, nil
}

type googleJSON struct {
	j *json.Decoder
}

func (pr *googleJSON) ReadPoint() (Point, error) {
	if !pr.j.More() {
		return Point{}, io.EOF
	}

	var p struct {
		TimestampMs string `json:"timestampMs"`

		LatE7  float64 `json:"latitudeE7"`
		LongE7 float64 `json:"longitudeE7"`

		Acc json.Number `json:"accuracy"`

		Alt  json.Number `json:"altitude"`
		VAcc json.Number `json:"verticalAccuracy"`
	}
	if err := pr.j.Decode(&p); err != nil {
		return Point{}, &DecodeError{err}
	}

	ms, err := strconv.ParseInt(p.TimestampMs, 0, 64)
	if err != nil {
		return Point{}, decodeError("invalid timestamp %v", p)
	}

	ts := time.Unix(ms/1000, (ms%1000)*1e6).UTC()
	pt := Point{
		Time: ts,
		Lat:  p.LatE7 / 1e7,
		Long: p.LongE7 / 1e7,
	}

	if v, err := p.Acc.Float64(); err == nil {
		pt.Acc = v
	} else {
		pt.Acc = NoAccuracy
	}
	if v, err := p.Alt.Float64(); err == nil {
		pt.Ele.Valid = true
		pt.Ele.Float64 = v
		if v, err := p.VAcc.Float64(); err == nil {
			pt.Ele.Acc = v
		} else {
			pt.Ele.Acc = NoAccuracy
		}
	}

	return pt, nil
}

func readTokens(j *json.Decoder, tokens ...json.Token) error {
	for _, w := range tokens {
		tok, err := j.Token()
		if err != nil {
			return err
		}
		if tok != w {
			return fmt.Errorf("Expected %v, got %v", w, tok)
		}
	}
	return nil
}
