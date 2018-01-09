package trackio

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

// isXML returns a DetectFunc for an xml having the specified docNodeName
func isXML(docNodeName string) DetectFunc {
	return func(p []byte) bool {
		d := xml.NewDecoder(bytes.NewReader(p))
		doc, err := nextStartElement(d)
		return err == nil && doc.Name.Local == docNodeName
	}
}

func nextStartElement(d *xml.Decoder) (se xml.StartElement, err error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se, nil
		}
	}
	panic("unreachable")
}

// errNoMore is returned when there are no more child elements in the parent xml set.
var errNoMore = errors.New("no more")

func nextChildElement(d *xml.Decoder, name string) (se xml.StartElement, err error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		switch tok.(type) {
		case xml.StartElement:
			if se.Name.Local == name {
				return se, nil
			}
		case xml.EndElement:
			return xml.StartElement{}, errNoMore
		}
	}
	panic("unreachable")
}

type xmlTreeOp int

const (
	xmlSkip   xmlTreeOp = iota // skip element
	xmlEnter                   // enter element
	xmlReturn                  // return current element as result
)

type xmlTreeFunc func(path []xml.Name, elem xml.Name) xmlTreeOp

type xmlTreeDecoder struct {
	d  *xml.Decoder
	fn xmlTreeFunc

	path []xml.Name
}

func newXMLTreeDecoder(d *xml.Decoder, fn xmlTreeFunc) *xmlTreeDecoder {
	return &xmlTreeDecoder{
		d:  d,
		fn: fn,
	}
}

func (x *xmlTreeDecoder) next() (xml.StartElement, error) {
	for {
		tok, err := x.d.Token()
		if err != nil {
			return xml.StartElement{}, err
		}

		switch e := tok.(type) {

		case xml.StartElement:
			switch x.fn(x.path, e.Name) {
			case xmlSkip:
				x.d.Skip()
			case xmlEnter:
				x.path = append(x.path, e.Name)
			case xmlReturn:
				return e, nil
			}

		case xml.EndElement:
			if len(x.path) == 0 {
				return xml.StartElement{}, io.EOF
			}
			x.path = x.path[:len(x.path)-1]
		}
	}
	panic("unreachable")
}

func wantXMLPath(path ...string) xmlTreeFunc {
	return xmlTreeFunc(func(p []xml.Name, e xml.Name) xmlTreeOp {
		d := len(p)
		if d < len(path) && e.Local == path[d] {
			if d == len(path)-1 {
				return xmlReturn
			} else {
				return xmlEnter
			}
		} else {
			return xmlSkip
		}
	})
}

func xmlCharData(d *xml.Decoder) (string, error) {
	buf := new(bytes.Buffer)
Loop:
	for {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		switch x := tok.(type) {
		case xml.CharData:
			buf.Write([]byte(x))
		case xml.StartElement:
			d.Skip()
		case xml.EndElement:
			break Loop
		}
	}
	return buf.String(), nil
}
