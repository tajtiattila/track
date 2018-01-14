package track

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"unsafe"
)

type Packer struct {
	pk Packed

	work Track

	bits []bitsPacker

	buf   []*Packed
	stack []*Packed
}

const packWindow = 512

func NewPacker() *Packer {
	k := &Packer{
		work: make([]Point, 0, packWindow),
		pk: Packed{
			acc: ptAcc{tacc: 100, dacc: 10},
		},
	}

	for nbytes := 2; nbytes <= 8; nbytes++ {
		k.bits = append(k.bits, bitPk(nbytes))
	}

	return k
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
	if len(k.work) == 0 {
		return
	}
	trk := k.work
	k.work = k.work[:0]
	k.stack = append(k.stack[:0], k.buf...)
	k.pk.append(k.recpack(trk))
}

func (k *Packer) recpack(trk Track) Packed {
	n := len(trk)
	pk := k.bitspack(trk)
	if n > 8 {
		n /= 2
		t1, t2 := trk[:n], trk[n:]
		pk1 := k.bitspack(t1)
		pk2 := k.bitspack(t2)
		if pk1.memsize()+pk2.memsize() < pk.memsize() {
			return pk1.append(pk2)
		}
	}
	return pk
}

func (k *Packer) bitspack(trk Track) Packed {
	var best *Packed
	for _, bp := range k.bits {
		pk := k.popPacked()
		pk.packpack(trk, &bp)
		if best == nil || pk.memsize() < best.memsize() {
			best = pk
		} else {
			k.pushPacked(pk)
		}
	}
	return *best
}

func (k *Packer) popPacked() *Packed {
	n := len(k.stack)
	if n > 0 {
		p := k.stack[n-1]
		k.stack = k.stack[:n-1]
		return p
	}
	p := &Packed{
		acc: k.pk.acc,
	}
	k.buf = append(k.buf, p)
	return p
}

func (k *Packer) pushPacked(p *Packed) {
	k.stack = append(k.stack, p)
}

type Packed struct {
	frame []relFrame
	pack  []byte

	acc ptAcc // accuracy reducer
}

func (k Packed) WriteTo(w io.Writer) (n int, err error) {
	buf := make([]byte, 256)
	buf[0] = 1 // version
	i := 1
	m := binary.PutUvarint(buf[i:], uint64(k.acc.tacc))
	i += m
	m = binary.PutUvarint(buf[i:], uint64(k.acc.dacc))
	i += m
	m = binary.PutUvarint(buf[i:], uint64(len(k.frame)))
	i += m
	m = binary.PutUvarint(buf[i:], uint64(len(k.pack)))
	i += m

	n, err = w.Write(buf[:i])
	if err != nil {
		return n, err
	}

	bo := binary.BigEndian
	for _, f := range k.frame {
		bo.PutUint64(buf, uint64(f.pt.t))
		i = 8

		bo.PutUint32(buf[i:], uint32(f.pt.lat))
		i += 4

		bo.PutUint32(buf[i:], uint32(f.pt.long))
		i += 4

		bo.PutUint64(buf[i:], uint64(f.data))
		i += 8

		m, err = w.Write(buf[:i])
		n += m
		if err != nil {
			return n, err
		}
	}

	m, err = w.Write(k.pack)
	n += m
	return n, err
}

func appendUvarint(dst []byte, v uint64) []byte {
	var buf [16]byte
	n := binary.PutUvarint(buf[:], v)
	return append(dst, buf[:n]...)
}

func (k *Packed) memsize() uintptr {
	nf := uintptr(len(k.frame)) * unsafe.Sizeof(relFrame{})
	np := uintptr(len(k.pack))
	return nf + np
}

type relFrame struct {
	pt   Point
	data uint64 // dbits, elemBytes and offset
}

func relFr(pt Point, ofs, elemBytes, dbits int) relFrame {
	d := uint64(ofs) |
		uint64(elemBytes)<<48 |
		uint64(dbits)<<56
	return relFrame{
		pt:   pt,
		data: d,
	}
}

const ofsMask = uint64(0xffffffffffff)

func (f relFrame) ofs() int {
	return int(f.data & ofsMask) // bottom 48 bits
}

func (f relFrame) elemBytes() int {
	return int(f.data>>48) & 0xff
}

func (f relFrame) dbits() uint {
	return uint(f.data>>56) & 0xff
}

func (f *relFrame) addOfs(ofs int) {
	ofs += f.ofs()
	f.data = (f.data & ^ofsMask) | uint64(ofs)
}

func (f *relFrame) setDbits(v uint) {
	f.data = f.data & ^(uint64(0xff) << 56)
	f.data |= uint64(v) << 56
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
	pkr := bitPk(bits)

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

func (pk *Packed) append(q Packed) Packed {
	n := len(pk.frame)
	d := len(pk.pack)
	pk.frame = append(pk.frame, q.frame...)
	pk.pack = append(pk.pack, q.pack...)
	for i := n; i < len(pk.frame); i++ {
		pk.frame[i].addOfs(d)
	}
	return *pk
}

func (pk *Packed) packpack(trk Track, bpk *bitsPacker) {
	start := 0
	for i, p0 := range trk {
		p := pk.acc.enc(p0)
		frame := true
		if i != 0 && bpk.add(p) {
			frame = false
		}

		if frame {
			if start < i {
				var dbits uint
				pk.pack, dbits = bpk.pack(pk.pack)
				lastFrame := &pk.frame[len(pk.frame)-1]
				lastFrame.setDbits(dbits)
			}
			start = i + 1

			bpk.frame(p)
			pk.frame = append(pk.frame, relFr(p, len(pk.pack), bpk.nbytes, 0))
		}
	}
}

type bitsPacker struct {
	nbytes int

	fr Point

	pt []relPt

	elem []relPackElem

	idx int // last valid elem idx
}

type relPackElem struct {
	tbits uint
	dbits uint
	valid bool
}

type relPt struct {
	t         uint64
	lat, long uint32
}

func bitPk(nbytes int) bitsPacker {
	p := bitsPacker{
		nbytes: nbytes,
	}

	const minBits = 4

	bits := 8 * nbytes
	imax := (bits - minBits) / 2
	for i := minBits; i <= imax; i++ {
		dbits := uint(i)
		tbits := uint(bits) - 2*dbits
		p.elem = append(p.elem, relPackElem{dbits: dbits, tbits: tbits})
	}
	return p
}

func (k *bitsPacker) last() (tbits, dbits uint) {
	if k.idx < 0 {
		panic("bitsPacker: illegal last")
	}
	e := k.elem[k.idx]
	return e.tbits, e.dbits
}

func (k *bitsPacker) frame(pt Point) {
	k.fr = pt
	k.pt = k.pt[:0]
	for i := range k.elem {
		k.elem[i].valid = true
	}
}

func (k *bitsPacker) add(pt Point) bool {
	dt := pt.t - k.fr.t
	if dt < 0 {
		panic("track invalid")
	}
	dlat := drelenc(pt.lat - k.fr.lat)
	dlong := drelenc(pt.long - k.fr.long)
	bt := uint(bits.Len64(uint64(dt)))
	bd := uint(bits.Len32(uint32(dlat)))
	if x := uint(bits.Len32(uint32(dlong))); x > bd {
		bd = x
	}

	hasvalid := false
	k.idx = -1
	for i := range k.elem {
		e := &k.elem[i]
		if e.valid {
			if k.idx == -1 {
				k.idx = i
			}
			if bt <= e.tbits && bd <= e.dbits {
				hasvalid = true
			} else {
				e.valid = false
			}
		}
	}

	if hasvalid {
		k.pt = append(k.pt, relPt{uint64(dt), dlat, dlong})
	}

	return hasvalid
}

func (k *bitsPacker) pack(dst []byte) ([]byte, uint) {
	if k.idx < 0 {
		panic("bitsPacker: illegal pack")
	}
	e := k.elem[k.idx]
	for _, d := range k.pt {
		v := ((uint64(d.t) << e.dbits) | uint64(d.lat)<<e.dbits) | uint64(d.long)

		for n := uint(k.nbytes); n >= 0; {
			n--
			shift := n * 8
			dst = append(dst, byte((v>>shift)&0xff))
		}
	}
	return dst, e.dbits
}

func drelenc(v int32) uint32 {
	return uint32((v << 1) ^ (v >> 31))
}

func dreldec(v uint32) int32 {
	x := int32(v >> 1)
	if v&1 != 0 {
		x = ^x
	}
	return x
}
