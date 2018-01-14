package track

import (
	"fmt"
	"math/bits"
	"unsafe"
)

type Packer struct {
	pk Packed

	work []Point
}

const packWindow = 512

func NewPacker() *Packer {
	return &Packer{
		work: make([]Point, 0, packWindow),
	}
}

func (k *Packer) Add(pt Point) {
	k.work = append(k.work, pt)

	if len(k.work) > packWindow {
		k.pack()
	}
}

func (k *Packer) Packed() Packed {
	k.pack()
	return k.pk
}

func (k *Packer) pack() {
	panic("TODO: implement")
}

type Packed struct {
	frame []relFrame
	pack  []byte
}

type relFrame struct {
	pt    Point
	ofsex uint64
}

func (f relFrame) ofs() int {
	return int(f.ofsex & 0xffffffffffff) // bottom 48 bits
}

func (f relFrame) elemBytes() int {
	return int(f.ofsex>>48) & 0xff
}

func (f relFrame) dbits() int {
	return int(f.ofsex>>56) & 0xff
}

type ptAcc struct {
	tacc int64
	dacc int32
}

func (k ptAcc) enc(pt Point) Point {
	pt = k.trunc(pt)
	return Point{
		pt.t / k.tacc,
		pt.lat / k.dacc,
		pt.long / k.dacc,
	}
}

func (k ptAcc) trunc(pt Point) Point {
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

func (k ptAcc) dec(pt Point) Point {
	return Point{
		pt.t * k.tacc,
		pt.lat * k.dacc,
		pt.long * k.dacc,
	}
}

func RelPack(trk Track) {
	nb := uintptr(len(trk)) * unsafe.Sizeof(Point{})
	fmt.Printf(" %d bytes = %.3fM\n", nb, float64(nb)/(1<<20))

	nu := uintptr(len(trk)) * unsafe.Sizeof(Point{})

	/*

		for i := 3; i < 8; i++ {
			bits := i * 8
			s := relPack(trk, bits)

			fmt.Printf(" %d pack: %d frames, %d bytes = %.3fM %.2f%%\n", bits,
				s.nframe, s.nbytes, float64(s.nbytes)/(1<<20),
				float64(s.nbytes)/float64(nu)*100)
		}

		for win := uint(3); win <= 12; win++ {
			window := int(1 << win)
			var s relPackStat
			for i := 0; i < len(trk); i += window {
				part := trk[i:]
				if len(part) > window {
					part = part[:window]
				}
				x := bestRelPack(part)
				s.nframe += x.nframe
				s.nbytes += x.nbytes
			}

			fmt.Printf(" %d window: %d frames, %d bytes = %.3fM %.2f%%\n", window,
				s.nframe, s.nbytes, float64(s.nbytes)/(1<<20),
				float64(s.nbytes)/float64(nu)*100)
		}

	*/

	const window = 256
	var s relPackStat
	for i := 0; i < len(trk); i += window {
		part := trk[i:]
		if len(part) > window {
			part = part[:window]
		}
		x := bestRelPackRec(part)
		s.nframe += x.nframe
		s.nbytes += x.nbytes
	}

	fmt.Printf(" %d frames, %d bytes = %.3fM %.2f%%\n",
		s.nframe, s.nbytes, float64(s.nbytes)/(1<<20),
		float64(s.nbytes)/float64(nu)*100)

	fmt.Println()
}

type relPackStat struct {
	bytesPerPack int
	nframe       int
	nbytes       int
}

func bestRelPackRec(trk Track) relPackStat {
	n := len(trk)
	s := bestRelPack(trk)
	if n > 8 {
		n /= 2
		t1, t2 := trk[:n], trk[n:]
		s1 := bestRelPackRec(t1)
		s2 := bestRelPackRec(t2)
		if s1.nbytes+s2.nbytes < s.nbytes {
			return relPackStat{
				nframe: s1.nframe + s2.nframe,
				nbytes: s1.nbytes + s2.nbytes,
			}
		}
	}
	return s
}

func bestRelPack(trk Track) relPackStat {
	var best relPackStat
	for i := 3; i < 8; i++ {
		bits := i * 8
		s := relPack(trk, bits)
		if i == 3 || s.nbytes < best.nbytes {
			best = s
		}
	}
	return best
}

func relPack(trk Track, bits int) relPackStat {
	pkr := newBitsPacker(bits)

	acc := ptAcc{tacc: 100, dacc: 10}

	var nframe, npk int

	for i, p0 := range trk {
		p := acc.enc(p0)
		frame := true
		if i != 0 && pkr.add(p) {
			npk++
			frame = false
		}

		if frame {
			pkr.frame(p)
			nframe++
		}
	}

	frameBytes := uintptr(nframe) * unsafe.Sizeof(relFrame{})
	packBytes := uintptr(npk * (bits / 8))
	return relPackStat{
		bytesPerPack: bits / 8,
		nframe:       nframe,
		nbytes:       int(frameBytes + packBytes),
	}
}

type bitsPacker struct {
	pt Point

	elem []relPackElem
}

type relPackElem struct {
	tbits uint
	dbits uint
	valid bool
}

func newBitsPacker(bits int) *bitsPacker {
	p := new(bitsPacker)
	const minBits = 4
	imax := (bits - minBits) / 2
	for i := minBits; i <= imax; i++ {
		dbits := uint(i)
		tbits := uint(bits) - 2*dbits
		p.elem = append(p.elem, relPackElem{dbits: dbits, tbits: tbits})
	}
	return p
}

func (k *bitsPacker) frame(pt Point) {
	k.pt = pt
	for i := range k.elem {
		k.elem[i].valid = true
	}
}

func (k *bitsPacker) add(pt Point) bool {
	dt := pt.t - k.pt.t
	if dt < 0 {
		panic("track invalid")
	}
	dlat := pt.lat - k.pt.lat
	dlong := pt.long - k.pt.long
	bt := uint(bits.Len64(uint64(dt)))
	if dlat < 0 {
		dlat = -dlat
	}
	if dlong < 0 {
		dlong = -dlong
	}
	bd := uint(bits.Len32(uint32(dlat)) + 1)
	if x := uint(bits.Len32(uint32(dlat)) + 1); x > bd {
		bd = x
	}

	hasvalid := false
	for i := range k.elem {
		e := &k.elem[i]
		if e.valid {
			if bt <= e.tbits && bd <= e.dbits {
				hasvalid = true
			} else {
				e.valid = false
			}
		}
	}

	return hasvalid
}
