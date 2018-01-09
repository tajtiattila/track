// Package trackio is a simple GPS track decoder.
//
// GPX, KML and Google location history JSON formats are supported
// by this package.
//
// Additional formats may be registered with RegisterFormat.
package trackio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("trackio: unknown format")

// DecodeError indicates that a point within a track file is invalid.
type DecodeError struct {
	Reason error
}

func (e *DecodeError) Error() string { return "trackio: " + e.Reason.Error() }

func decodeError(format string, a ...interface{}) error {
	return &DecodeError{fmt.Errorf(format, a...)}
}

// DetectFormat determines if data represents a track file.
//
// The string returned the name of the format.
//
// Data should be large enough to contain the
// first data element (such as the document node in XML).
// Typically the first few kilobytes is sufficient.
func DetectFormat(data []byte) (string, bool) {
	f, ok := detectFormat(data)
	return f.name, ok
}

func detectFormat(data []byte) (format, bool) {
	for _, f := range formats {
		if f.detect(data) {
			return f, true
		}
	}
	return format{}, false
}

// DefaultAccuracy is the default accuracy set by NewDecoder.
const DefaultAccuracy = 200

// Decoder decodes tracks from an input stream.
type Decoder struct {
	// PointReader used to decode Tracks.
	//
	// Clients may use PointReader directly
	// to decode individual track points.
	PointReader

	// Decoder.Track uses Accuracy to filter track points.
	// Track points with higher accuracy values are ignored.
	//
	// An Accuracy >= NoAccuracy means no minimum accuracy.
	//
	// Formats that don't have accuracy information (such as KML)
	// does not use this value.
	Accuracy float64

	// Decoder.Track calls HandleDecodeError to
	// handle track point decoding errors.
	//
	// When not nil, decode errors are filtered with it.
	// If HandleDecodeError returns nil,
	// track point decoding errors are ignored.
	HandleDecodeError func(*DecodeError) error

	hasAccuracy bool // has point with Acc < NoAccuracy
}

// NewDecoder returns a new decoder that reads from r
// with Accuracy set to DefaultAccuracy.
func NewDecoder(r io.Reader) *Decoder {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, io.LimitReader(r, 64<<10))
	if err != nil {
		return newErrDecoder(err)
	}

	f, ok := detectFormat(buf.Bytes())
	if !ok {
		return newErrDecoder(ErrFormat)
	}

	pr, err := f.newpr(io.MultiReader(buf, r))
	if err != nil {
		return newErrDecoder(err)
	}

	return newDecoder(pr)
}

func newDecoder(pr PointReader) *Decoder {
	return &Decoder{
		PointReader: pr,
		Accuracy:    DefaultAccuracy,
	}
}

func newErrDecoder(err error) *Decoder {
	return newDecoder(&errPointReader{err})
}

type errPointReader struct {
	err error
}

func (r *errPointReader) ReadPoint() (Point, error) { return Point{}, r.err }

// Point returns the next track point from the underlying PointReader.
//
// The return value reset is true when then
// the first track point with a valid accuracy value is encountered,
// and therefore the client should throw away point(s) decoded so far.
//
// It returns only track points with better than d.Accuracy.
//
// If there is no track point with a valid accuracy value,
// all points are returned.
func (d *Decoder) Point() (pt Point, reset bool, err error) {
	for {
		pt, err = d.ReadPoint()
		if err != nil {
			if err == io.EOF {
				return Point{}, false, err
			}
			err = d.handleError(err)
			if err == nil {
				continue
			}
			return Point{}, false, err
		}

		// first point with valid accuracy value
		if !d.hasAccuracy && pt.Acc < NoAccuracy {
			d.hasAccuracy = true
			if d.Accuracy < NoAccuracy {
				reset = true
			}
		}

		if !d.hasAccuracy || pt.Acc <= d.Accuracy {
			return pt, reset, err
		}
	}
	panic("unreachable")
}

// Track returns the track points in
// chronological order from the underlying PointReader.
//
// It returns only track points with better than d.Accuracy.
//
// If there is no track point with a valid accuracy value,
// all points are returned.
func (d *Decoder) Track() (Track, error) {
	var trk Track
	for {
		p, reset, err := d.Point()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			trk.Sort()
			return trk, err
		}

		if reset {
			trk = trk[:0]
		}

		trk = append(trk, p)
	}
	panic("unreachable")
}

func (d *Decoder) handleError(err error) error {
	if d.HandleDecodeError == nil {
		return err
	}

	if de, ok := err.(*DecodeError); ok {
		err = d.HandleDecodeError(de)
	}

	return err
}

// PointReader reads raw track points from the underlying input source.
//
// Its ReadPoint method is safe to use
// after an error of type *DecodeError was returned
// to read further track points.
type PointReader interface {
	ReadPoint() (Point, error)
}

// DetectFunc detects if the p represents a valid prefix
// (first bytest of the underlying reader)
// for the current format.
type DetectFunc func(p []byte) bool

// NewPointReader creates a new PointReader reading from r.
type NewPointReader func(r io.Reader) (PointReader, error)

// RegisterFormat registers a new track file format.
func RegisterFormat(
	name string,
	detect DetectFunc,
	newPointReader NewPointReader,
) {

	for _, f := range formats {
		if f.name == name {
			panic("name already registered")
		}
	}
	formats = append(formats, format{name, detect, newPointReader})
}

type format struct {
	name   string
	detect DetectFunc
	newpr  NewPointReader
}

var formats []format
