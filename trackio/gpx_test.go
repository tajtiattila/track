package trackio_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/tajtiattila/track/trackio"
)

// https://en.wikipedia.org/wiki/GPS_Exchange_Format
var sampleGPX = `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>

<gpx xmlns="http://www.topografix.com/GPX/1/1" xmlns:gpxx="http://www.garmin.com/xmlschemas/GpxExtensions/v3" xmlns:gpxtpx="http://www.garmin.com/xmlschemas/TrackPointExtension/v1" creator="Oregon 400t" version="1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd http://www.garmin.com/xmlschemas/GpxExtensions/v3 http://www.garmin.com/xmlschemas/GpxExtensionsv3.xsd http://www.garmin.com/xmlschemas/TrackPointExtension/v1 http://www.garmin.com/xmlschemas/TrackPointExtensionv1.xsd">
  <metadata>
    <link href="http://www.garmin.com">
      <text>Garmin International</text>
    </link>
    <time>2009-10-17T22:58:43Z</time>
  </metadata>
  <trk>
    <name>Example GPX Document</name>
    <trkseg>
      <trkpt lat="47.644548" lon="-122.326897">
        <ele>4.46</ele>
        <time>2009-10-17T18:37:26Z</time>
      </trkpt>
      <trkpt lat="47.644548" lon="-122.326897">
        <ele>4.94</ele>
        <time>2009-10-17T18:37:31Z</time>
      </trkpt>
      <trkpt lat="47.644548" lon="-122.326897">
        <ele>6.87</ele>
        <time>2009-10-17T18:37:34Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>
`

func TestGPX(t *testing.T) {
	r := bytes.NewReader([]byte(sampleGPX))
	trk, err := trackio.NewDecoder(r).Track()
	if err != nil {
		t.Fatal(err)
	}

	const wantLen = 3
	if len(trk) != wantLen {
		t.Fatalf("track length mismatch: want %d got %d", wantLen, len(trk))
	}

	pointEqual(t, trk[0], trackio.Pt(
		time.Date(2009, 10, 17, 18, 37, 26, 0, time.UTC),
		47.644548,
		-122.326897,
	))
}
