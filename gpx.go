package track

import (
	"encoding/xml"
	"io"
	"time"
)

func init() {
	registerFormat("gpx", isXML("gpx"), decodeGPX)
}

/* GPX Format:

<?xml .. ?>
<gpx version="1.0" ..>
	<trk>
		<trkseg>
			<trkpt lat="45.51832616" lon="17.85887513"><time>2015-05-03T10:16:58Z</time></trkpt>
		</trkseg>
		...

*/

func decodeGPX(r io.Reader) (Track, error) {
	d := xml.NewDecoder(r)

	var gpx gpxData
	if err := d.Decode(&gpx); err != nil {
		return nil, &DecodeError{err}
	}

	t := make(Track, len(gpx.Pt))
	for i, pt := range gpx.Pt {
		ts, err := time.Parse(time.RFC3339, pt.Time)
		if err != nil {
			return nil, decodeError("invalid timestamp %q", pt.Time)
		}
		t[i] = Pt(ts, pt.Lat, pt.Long)
	}
	return t, nil
}

type gpxData struct {
	XMLName xml.Name `xml:"gpx"`
	Pt      []gpxPt  `xml:"trk>trkseg>trkpt"`
}

type gpxPt struct {
	Lat  float64 `xml:"lat,attr"`
	Long float64 `xml:"lon,attr"`
	Time string  `xml:"time"`
}
