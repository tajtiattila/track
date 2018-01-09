package trackio

import (
	"encoding/xml"
	"io"
	"strconv"
	"time"
)

func init() {
	RegisterFormat("gpx", isXML("gpx"), newGPX)
}

/* GPX Format:

<?xml .. ?>
<gpx version="1.0" ..>
	<trk>
		<trkseg>
			<trkpt lat="45.51832616" lon="17.85887513"><time>2015-05-03T10:16:58Z</time></trkpt>
		</trkseg>
		...

see http://www.topografix.com/GPX/1/1

*/

func newGPX(r io.Reader) (PointReader, error) {
	d := xml.NewDecoder(r)

	doc, err := nextStartElement(d)
	if err != nil {
		panic("impossible")
	}

	if doc.Name.Local != "gpx" {
		panic("impossible")
	}

	return &gpx{
		td: newXMLTreeDecoder(d, wantXMLPath("trk", "trkseg", "trkpt")),
	}, nil
}

type gpx struct {
	td *xmlTreeDecoder
}

func (g *gpx) ReadPoint() (Point, error) {
	se, err := g.td.next()
	if err != nil {
		return Point{}, err
	}
	return g.decodePt(se)
}

func (g *gpx) decodePt(se xml.StartElement) (Point, error) {
	var p gpxPt
	err := g.td.d.DecodeElement(&p, &se)
	if err != nil {
		return Point{}, err
	}

	ts, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return Point{}, decodeError("invalid timestamp %q", p.Time)
	}

	pt := Point{
		Time: ts,
		Lat:  p.Lat,
		Long: p.Long,
	}

	pt.Acc = gpsAccuracy(&p, true)

	if v, err := strconv.ParseFloat(p.Ele, 64); err == nil {
		pt.Ele.Valid = true
		pt.Ele.Float64 = v
		pt.Ele.Acc = gpsAccuracy(&p, false)
	}

	return pt, nil
}

type gpxPt struct {
	Lat  float64 `xml:"lat,attr"`
	Long float64 `xml:"lon,attr"`
	Time string  `xml:"time"`
	Ele  string  `xml:"ele"`
	Src  string  `xml:"src"`
	Sat  int     `xml:"sat"`
	HDOP string  `xml:"hdop"`
	VDOP string  `xml:"vdop"`
	PDOP string  `xml:"pdop"`
}

// Values and the logic below is highly speculative.

const networkAccuracy = 3e3 // meters
const baseGPSAccuracy = 5   // meters

func gpsAccuracy(g *gpxPt, horz bool) float64 {
	if g.Src == "network" {
		return networkAccuracy
	}
	var dops string
	if horz {
		dops = g.HDOP
	} else {
		dops = g.VDOP
	}
	if dop, err := strconv.ParseFloat(dops, 64); err == nil {
		return baseGPSAccuracy * dop
	}
	return baseGPSAccuracy
}
