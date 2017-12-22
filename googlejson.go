package track

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
)

/* Google location history JSON format:

{"locations": [
  { "timestampMs": "1513838533570", "latitudeE7": 461964457, "longitudeE7": 182726693, ... },
  { "timestampMs": ...}
]}

*/

func decodeGoogleJSON(r io.Reader) (Track, error) {
	j := json.NewDecoder(r)
	if err := readTokens(j,
		json.Delim('{'),
		"locations",
		json.Delim('[')); err != nil {

		return nil, errBadFormat
	}

	var t Track
	var pt struct {
		TimestampMs string  `json:"timestampMs"`
		LatE7       float64 `json:"latitudeE7"`
		LongE7      float64 `json:"longitudeE7"`
	}
	for j.More() {
		if err := j.Decode(&pt); err != nil {
			return nil, decodeError("GoogleJSON: decode err %v", err.Error())
		}

		ms, err := strconv.ParseInt(pt.TimestampMs, 0, 64)
		if err != nil {
			return nil, decodeError("GoogleJSON: invalid timestamp %v", pt)
		}
		ts := time.Unix(int64(ms)/1000, (int64(ms)%1000)*1e6).UTC()

		t = append(t, Point{ts, pt.LatE7 / 1e7, pt.LongE7 / 1e7})
	}
	return t, nil
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
