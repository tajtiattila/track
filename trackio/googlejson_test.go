package trackio_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/tajtiattila/track/trackio"
)

var sampleGoogleJSON = `{
  "locations" : [ {
    "timeStampMs": "1223019649000",
    "latitudeE7": 523190516,
    "longitudeE7": 94098216
  }, {
    "timeStampMs": "1223019756000",
    "latitudeE7": 523178066,
    "longitudeE7": 94103316
  }, {
    "timeStampMs": "1223019840000",
    "latitudeE7": 523168933,
    "longitudeE7": 94106649
  }, {
    "timeStampMs": "1223019853000",
    "latitudeE7": 523168066,
    "longitudeE7": 94106666
  }, {
    "timeStampMs": "1223019977000",
    "latitudeE7": 523154899,
    "longitudeE7": 94110116
  }, {
    "timeStampMs": "1223020097000",
    "latitudeE7": 523141699,
    "longitudeE7": 94116666
  }, {
    "timeStampMs": "1223020216000",
    "latitudeE7": 523128866,
    "longitudeE7": 94114933,
    "activity" : [ {
      "timestampMs" : "1513838479184",
      "activity" : [ {
        "type" : "ON_FOOT",
        "confidence" : 100
      }, {
        "type" : "WALKING",
        "confidence" : 100
      } ]
	} ]
  }, {
    "timeStampMs": "1223020295000",
    "latitudeE7": 523121066,
    "longitudeE7": 94110383
  }, {
    "timeStampMs": "1223020384000",
    "latitudeE7": 523112833,
    "longitudeE7": 94105183
  }, {
    "timeStampMs": "1223020449000",
    "latitudeE7": 523111866,
    "longitudeE7": 94104550
  }, {
    "timeStampMs": "1223020462000",
    "latitudeE7": 523111699,
    "longitudeE7": 94104600
  }, {
    "timeStampMs": "1223020547000",
    "latitudeE7": 523112449,
    "longitudeE7": 94103550
  }, {
    "timeStampMs": "1223020671000",
    "latitudeE7": 523105049,
    "longitudeE7": 94102733
  }, {
    "timeStampMs": "1223020736000",
    "latitudeE7": 523098833,
    "longitudeE7": 94099599
  }, {
    "timeStampMs": "1223020844000",
    "latitudeE7": 523091833,
    "longitudeE7": 94081433
  }, {
    "timeStampMs": "1223020852000",
    "latitudeE7": 523091899,
    "longitudeE7": 94081299
  }, {
    "timeStampMs": "1223020880000",
    "latitudeE7": 523090666,
    "longitudeE7": 94075199
  }, {
    "timeStampMs": "1223020892000",
    "latitudeE7": 523090299,
    "longitudeE7": 94073766
  } ]
}`

func TestGoogleJSON(t *testing.T) {
	r := bytes.NewReader([]byte(sampleGoogleJSON))
	trk, err := trackio.NewDecoder(r).Track()
	if err != nil {
		t.Fatal(err)
	}

	const wantLen = 18
	if len(trk) != wantLen {
		t.Fatalf("track length mismatch: want %d got %d", wantLen, len(trk))
	}

	ts, _ := time.Parse(time.RFC3339, "2008-10-03T07:40:49Z")

	pointEqual(t, trk[0], trackio.Pt(ts, 52.3190516, 9.4098216))
}
