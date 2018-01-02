package track

import (
	"encoding/binary"
)

type Packed struct {
	elem []elem
	pack []byte

	tacc int64 // time accuracy
	dacc int64 // location accuracy
}

type elem struct {
	pt  Point
	ofs int // offset into parent pack slice
}

type PackStat struct {
	NumPacked int // number of packed points
	MemSize   int // total number of bytes

	Elem []struct {
		Bytes int // length of single element in bytes
		Count int // count of packed elements with this length
	}
}

func Pack(trk Track, tacc, dacc, packlen int) (Packed, PackStat) {
	k := Packed{
		tacc: int64(tacc),
		dacc: int64(dacc),
	}
	tx := k.tacc / 2

	// packd is used to convert non-packed lat/long
	// to packed representation
	var packd func(latlong int32) int64
	if dacc > 1 {
		dx := k.dacc / 2
		packd = func(latlong int32) int64 {
			x := int64(latlong)
			if x >= 0 {
				x += dx
			} else {
				x -= dx
			}
			return x / k.dacc
		}
	} else {
		packd = func(latlong int32) int64 { return int64(latlong) }
	}

	var lastpt Point
	var buf [64]byte
	var npacked int
	nlen := make([]int, 64)
	for i, pt := range trk {
		addpk := true
		if i != 0 {
			p := buf[:]

			dt := pt.t - lastpt.t
			dlat := pt.lat - lastpt.lat
			dlong := pt.long - lastpt.long

			n := binary.PutUvarint(p, uint64((dt+tx)/k.tacc))
			n += binary.PutVarint(p[n:], packd(dlat))
			n += binary.PutVarint(p[n:], packd(dlong))
			p = p[:n]

			e := k.elem[len(k.elem)-1]
			l := len(k.pack) - e.ofs
			if l+n < packlen {
				npacked++
				k.pack = append(k.pack, p...)
				addpk = false
				nlen[len(p)]++
			}
		}

		if addpk {
			k.elem = append(k.elem, elem{
				pt:  pt,
				ofs: len(k.pack),
			})
		}

		lastpt = pt
	}

	// stat
	s := PackStat{
		NumPacked: npacked,
		MemSize:   24*len(k.elem) + len(k.pack),
	}
	for i, n := range nlen {
		if n != 0 {
			s.Elem = append(s.Elem, struct{ Bytes, Count int }{
				Bytes: i,
				Count: n,
			})
		}
	}
	return k, s
}

func (k Packed) Unpack(dst Track) Track {
	k.ForEach(func(pt Point) error {
		dst = append(dst, pt)
		return nil
	})
	return dst
}

func (k Packed) ForEach(f func(pt Point) error) error {
	for i, e := range k.elem {
		if err := f(e.pt); err != nil {
			return err
		}

		var pack []byte
		if j := i + 1; j < len(k.elem) {
			next := k.elem[j]
			pack = k.pack[e.ofs:next.ofs]
		} else {
			pack = k.pack[e.ofs:]
		}

		for len(pack) > 0 {
			dt, n := binary.Uvarint(pack)
			if n <= 0 {
				panic("impossible")
			}
			pack = pack[n:]

			dlat, n := binary.Varint(pack)
			if n <= 0 {
				panic("impossible")
			}
			pack = pack[n:]

			dlong, n := binary.Varint(pack)
			if n <= 0 {
				panic("impossible")
			}
			pack = pack[n:]

			if err := f(Point{
				t:    e.pt.t + int64(dt)*k.tacc,
				lat:  e.pt.lat + int32(dlat*k.dacc),
				long: e.pt.long + int32(dlong*k.dacc),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
