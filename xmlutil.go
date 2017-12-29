package track

import (
	"bytes"
	"encoding/xml"
	"io"
)

// isXML tests if r is an xml having the specifier docNode
func isXML(r io.Reader, docNode string) (io.Reader, bool) {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, io.LimitReader(r, 64<<10))
	if err != nil {
		return nil, false
	}

	d := xml.NewDecoder(bytes.NewReader(buf.Bytes()))
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, false
		}
		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == docNode {
				return io.MultiReader(buf, r), true
			}
		}
	}
	panic("unreachable")
}
