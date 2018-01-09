package trackio

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

func init() {
	RegisterFormat("kml", isXML("kml"), newKML)
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

// kml: 0 or 1 «Feature»
// «Feature»: Document|Folder|Placemark
// Document: 0 or more «Feature» elements
// Folder: 0 or more «Feature» elements
// Placemark: 0 or more gx:Track elements

func newKML(r io.Reader) (PointReader, error) {
	d := xml.NewDecoder(r)

	doc, err := nextStartElement(d)
	if err != nil {
		panic("impossible")
	}

	if doc.Name.Local != "kml" {
		panic("impossible")
	}

	pr := new(kml)
	pr.td = newXMLTreeDecoder(d, xmlTreeFunc(pr.kmlTreeFunc))
	return pr, nil
}

type kml struct {
	td *xmlTreeDecoder

	nextErr error

	coord strFifo
	when  strFifo
}

func (k *kml) kmlTreeFunc(p []xml.Name, e xml.Name) xmlTreeOp {
	// TODO: check namespace
	if len(p) != 0 {
		last := p[len(p)-1]
		if last.Local == "Track" {
			if e.Local == "when" || e.Local == "coord" {
				return xmlReturn
			} else {
				return xmlSkip
			}
		}
		if last.Local == "Placemark" {
			if e.Local == "Track" {
				k.nextErr = k.checkLengthMismatch()
				k.coord.reset()
				k.when.reset()
				return xmlEnter
			} else {
				return xmlSkip
			}
		}
	}
	switch e.Local {
	case "Document", "Folder", "Placemark":
		return xmlEnter
	}
	return xmlSkip
}

func (k *kml) checkLengthMismatch() error {
	if k.coord.n != k.when.n {
		return decodeError("length mismatch (coord: %d, when: %d)",
			k.coord.n, k.when.n)
	}
	return nil
}

func (k *kml) ReadPoint() (Point, error) {
	p, err := k.readPoint()
	if err == io.EOF {
		if xerr := k.checkLengthMismatch(); xerr != nil {
			return Point{}, xerr
		}
	}
	return p, err
}

func (k *kml) readPoint() (Point, error) {
	for {
		if err := k.popErr(); err != nil {
			return Point{}, err
		}

		se, err := k.td.next()
		if err != nil {
			return Point{}, k.pushErr(err)
		}

		charData, err := xmlCharData(k.td.d)
		if err != nil {
			return Point{}, k.pushErr(err)
		}

		switch se.Name.Local {
		case "coord":
			k.coord.push(charData)
		case "when":
			k.when.push(charData)
		default:
			panic("impossible")
		}

		if err := k.popErr(); err != nil {
			return Point{}, err
		}

		if k.coord.has() && k.when.has() {
			coord, when := k.coord.pop(), k.when.pop()

			var lat, long, ele float64
			lat, long, ele, err = decodeKMLCoord(coord)
			if err != nil {
				return Point{}, err
			}

			ts, err := time.Parse(time.RFC3339, when)
			if err != nil {
				return Point{}, decodeError("invalid timestamp %q", when)
			}

			return Point{
				Time: ts.UTC(),
				Lat:  lat,
				Long: long,

				Acc: NoAccuracy,

				Ele: Elevation{
					Valid:   true, // NOTE: unknown if elevation is truly valid
					Float64: ele,
					Acc:     NoAccuracy,
				},
			}, nil
		}
	}
}

/*
		var c kmlCoord
		var garbage string
		// TODO(tajtiattila): check for garbage after lat/long?
		_, err := fmt.Sscanf(charData, "%f %f %f%s", &c.lat, &c.long, &c.ele, &garbage)

		// store coord even if there was an error to match up counts
		k.coord = append(k.coord, c)

		if err != nil {
			return Point{}, decodeError("invalid coord %q", charData)
		} else if garbage != "" {
			return Point{}, decodeError("garbage after coord %q", charData)
		}

	case "when":
		ts, err := time.Parse(time.RFC3339, when)

		// store when even if there was an error to match up counts
		k.when = append(k.when, ts)

		if err != nil {
			return Point{}, decodeError("invalid timestamp %q", when)
		}

	default:
		panic("impossible")
	}
*/

func (k *kml) popErr() error {
	err := k.nextErr
	k.nextErr = nil
	return err
}

func (k *kml) pushErr(err error) error {
	if xerr := k.nextErr; xerr != nil {
		k.nextErr = err
		return xerr
	}
	return err
}

func decodeKMLCoord(coord string) (lat, long, ele float64, err error) {
	var garbage string
	// TODO(tajtiattila): check for garbage after lat/long?
	n, err := fmt.Sscanf(coord, "%f %f %f%s", &lat, &long, &ele, &garbage)

	if n == 3 && err == io.EOF {
		return lat, long, ele, nil
	}

	if n == 4 {
		return 0, 0, 0, decodeError("garbage after coord %q", coord)
	}

	return 0, 0, 0, decodeError("invalid coord %q", coord)
}

func decodeKML(r io.Reader) (Track, error) {
	var kml kmlData
	if err := xml.NewDecoder(r).Decode(&kml); err != nil {
		return nil, &DecodeError{err}
	}

	var t []Point
	for _, k := range kml.Trk {
		seg, err := decodeKMLTrk(k)
		if err != nil {
			return nil, err
		}
		t = append(t, seg...)
	}
	return t, nil
}

func decodeKMLTrk(k kmlTrk) ([]Point, error) {

	if len(k.When) != len(k.Coord) {
		return nil, decodeError("length mismatch (when: %d, coord: %d)", len(k.When), len(k.Coord))
	}

	t := make([]Point, len(k.When))
	for i := range k.When {

		when := k.When[i]
		ts, err := time.Parse(time.RFC3339, when)
		if err != nil {
			return nil, decodeError("invalid timestamp %q", when)
		}

		coord := k.Coord[i]
		var lat, long, ele float64
		// TODO(tajtiattila): check for garbage after lat/long?
		if _, err := fmt.Sscanf(coord, "%f %f %f", &lat, &long, &ele); err != nil {
			return nil, decodeError("invalid coord %q", coord)
		}

		t[i] = Point{
			Time: ts.UTC(),
			Lat:  lat,
			Long: long,

			Acc: NoAccuracy,

			Ele: Elevation{
				Valid:   true,
				Float64: ele,
				Acc:     NoAccuracy,
			},
		}
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

type strFifo struct {
	buf  []string
	r, w int

	n int // total elements pushed
}

func (f *strFifo) has() bool {
	return f.r != f.w
}

func (f *strFifo) len() int {
	if f.r <= f.w {
		return f.w - f.r
	}
	return f.w + len(f.buf) - f.r
}

func (f *strFifo) push(s string) {
	f.n++
	if len(f.buf) == 0 {
		f.buf = make([]string, 100)
		f.buf[0] = s
		f.r, f.w = 0, 1
		return
	}

	f.buf[f.w] = s
	f.w = f.next(f.w)
	if f.w == f.r {
		// buffer full
		nbuf := make([]string, len(f.buf)*2)
		n := copy(nbuf, f.buf[f.r:])
		copy(nbuf[n:], f.buf[:f.w])

		f.r = 0
		f.w = len(f.buf)
		f.buf = nbuf
	}
}

func (f *strFifo) pop() string {
	if f.r == f.w {
		panic("invalid pop")
	}
	s := f.buf[f.r]
	f.r = f.next(f.r)
	if f.r == f.w {
		// empty buffer
		f.r, f.w = 0, 0
	}
	return s
}

func (f *strFifo) reset() {
	f.n, f.r, f.w = 0, 0, 0
}

func (f *strFifo) next(i int) int {
	i++
	if i < len(f.buf) {
		return i
	}
	return 0
}
