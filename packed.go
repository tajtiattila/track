package track

import (
	"encoding/binary"
	"sort"
	"time"

	"github.com/tajtiattila/track/internal/trackmath"
)

type Packed struct {
	frame []frame
	pack  []byte

	tacc int64 // time accuracy
	dacc int32 // location accuracy
}

type frame struct {
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
	if len(trk) == 0 {
		return Packed{}, PackStat{}
	}

	k := Packed{
		tacc: int64(tacc),
		dacc: int32(dacc),
	}
	trunc := func(pt Point) Point {
		t := pt.t + k.tacc/2
		t -= t % k.tacc

		lat := pt.lat
		if lat >= 0 {
			lat += k.dacc / 2
		} else {
			lat -= k.dacc / 2
		}
		lat -= lat % k.dacc

		long := pt.long
		if long >= 0 {
			long += k.dacc / 2
		} else {
			long -= k.dacc / 2
		}
		long -= long % k.dacc

		return Point{t, lat, long}
	}

	var lastpt Point
	var buf [64]byte
	var npacked int
	nlen := make([]int, 64)
	for i := range trk[:len(trk)-1] {
		pt := trunc(trk[i])
		addpk := true
		if i != 0 {
			p := buf[:]

			dt := pt.t - lastpt.t
			dlat := pt.lat - lastpt.lat
			dlong := pt.long - lastpt.long

			n := binary.PutUvarint(p, uint64(dt/k.tacc))
			n += binary.PutVarint(p[n:], int64(dlat/k.dacc))
			n += binary.PutVarint(p[n:], int64(dlong/k.dacc))
			p = p[:n]

			e := k.frame[len(k.frame)-1]
			l := len(k.pack) - e.ofs
			if l+n < packlen {
				npacked++
				k.pack = append(k.pack, p...)
				addpk = false
				nlen[len(p)]++
			}
		}

		if addpk {
			k.frame = append(k.frame, frame{
				pt:  pt,
				ofs: len(k.pack),
			})
		}

		lastpt = pt
	}

	// last point
	k.frame = append(k.frame, frame{
		pt:  trk[len(trk)-1],
		ofs: len(k.pack),
	})

	// stat
	s := PackStat{
		NumPacked: npacked,
		MemSize:   24*len(k.frame) + len(k.pack),
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
	for i, e := range k.frame {
		if err := f(e.pt); err != nil {
			return err
		}

		fu := k.unpackFrame(i)

		for !fu.done() {
			if err := f(fu.next()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Packed) At(t time.Time) (lat, long float64) {
	tt := itime(t)
	i := sort.Search(len(k.frame), func(i int) bool {
		return tt < k.frame[i].pt.t
	})

	if i == 0 {
		if len(k.frame) == 0 {
			return 0, 0
		}
		p := k.frame[0].pt
		return p.Lat(), p.Long()
	} else if i == len(k.frame) {
		p := k.frame[len(k.frame)-1].pt
		return p.Lat(), p.Long()
	}

	fu := k.unpackFrame(i - 1)

	for !fu.done() {
		p := fu.pt
		q := fu.next()
		if tt < q.t {
			return interp(t, p, q)
		}
	}

	return interp(t, fu.pt, k.frame[i].pt)
}

func interp(t time.Time, p, q Point) (lat, long float64) {
	return trackmath.Interpolate(t,
		p.Time(), p.Lat(), p.Long(),
		q.Time(), q.Lat(), q.Long())
}

func (k *Packed) unpackFrame(i int) frameUnpack {
	f := k.frame[i]
	fu := frameUnpack{
		tacc: k.tacc,
		dacc: k.dacc,
		pt:   f.pt,
	}
	if j := i + 1; j < len(k.frame) {
		next := k.frame[j]
		fu.pack = k.pack[f.ofs:next.ofs]
	} else {
		fu.pack = k.pack[f.ofs:]
	}
	return fu
}

type frameUnpack struct {
	tacc int64 // time accuracy
	dacc int32 // location accuracy

	pt   Point
	pack []byte
}

func (fu *frameUnpack) done() bool {
	return len(fu.pack) == 0
}

func (fu *frameUnpack) next() Point {
	dt, n := binary.Uvarint(fu.pack)
	if n <= 0 {
		panic("impossible")
	}
	fu.pack = fu.pack[n:]

	dlat, n := binary.Varint(fu.pack)
	if n <= 0 {
		panic("impossible")
	}
	fu.pack = fu.pack[n:]

	dlong, n := binary.Varint(fu.pack)
	if n <= 0 {
		panic("impossible")
	}
	fu.pack = fu.pack[n:]

	fu.pt = Point{
		t:    fu.pt.t + int64(dt)*fu.tacc,
		lat:  fu.pt.lat + int32(dlat)*fu.dacc,
		long: fu.pt.long + int32(dlong)*fu.dacc,
	}

	return fu.pt
}
