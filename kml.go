package track

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

func init() {
	registerFormat("kml", isXML("kml"), decodeKML)
}

/* KML Format:

<?xml .. ?>
<kml>
	<Document>
		<Placemark>
			<gx:Track>
				<altitudeMode>clampToGround</altitudeMode>
				<when>2017-12-21T08:44:28Z</when>
				<gx:coord>18.2726462 46.196407699999995 0</gx:coord>
...

*/

func decodeKML(r io.Reader) (Track, error) {
	var kml kmlData
	if err := xml.NewDecoder(r).Decode(&kml); err != nil {
		return nil, &DecodeError{err}
	}

	var t Track
	for _, k := range kml.Trk {
		seg, err := decodeKMLTrk(k)
		if err != nil {
			return nil, err
		}
		t = append(t, seg...)
	}
	return t, nil
}

func decodeKMLTrk(k kmlTrk) (Track, error) {

	if len(k.When) != len(k.Coord) {
		return nil, decodeError("length mismatch (when: %d, coord: %d)", len(k.When), len(k.Coord))
	}

	t := make(Track, len(k.When))
	for i := range k.When {

		when := k.When[i]
		ts, err := time.Parse(time.RFC3339, when)
		if err != nil {
			return nil, decodeError("invalid timestamp %q", when)
		}

		coord := k.Coord[i]
		var lat, long float64
		// TODO(tajtiattila): check for garbage after lat/long?
		if _, err := fmt.Sscanf(coord, "%f %f", &lat, &long); err != nil {
			return nil, decodeError("invalid coord %q", coord)
		}

		t[i] = Pt(ts, lat, long)
	}
	return t, nil
}

type kmlData struct {
	XMLName xml.Name `xml:"kml"`
	Trk     []kmlTrk `xml:"Document>Placemark>Track"`
}

type kmlTrk struct {
	When  []string `xml:"when"`
	Coord []string `xml:"coord"`
}
