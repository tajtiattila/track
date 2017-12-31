package track

import (
	"bytes"
	"encoding/xml"
)

// isXML returns a detectFunc for an xml having the specified docNode
func isXML(docNode string) detectFunc {
	return func(p []byte) bool {
		d := xml.NewDecoder(bytes.NewReader(p))
		for {
			tok, err := d.Token()
			if err != nil {
				return false
			}
			if se, ok := tok.(xml.StartElement); ok {
				if se.Name.Local == docNode {
					return true
				}
			}
		}
		panic("unreachable")
	}
}
