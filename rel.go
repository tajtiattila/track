package track

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"sort"
	"time"
	"unsafe"

	"github.com/tajtiattila/track/internal/trackmath"
)

func Pack(trk Track, tacc, dacc, baz int) (Packed, error) {
	r := NewPacker()
	for _, p := range trk {
		r.Add(p)
	}
	return r.Packed(), nil
}

type Packer struct {
	pk Packed

	work Track

	bits []*bitsPacker

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
		k.bits = append(k.bits, newBitsPacker(nbytes))
	}

	return k
}

func (k *Packer) Add(pt Point) {
	k.work = append(k.work, pt)

	if len(k.work) >= packWindow {
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
	pk := k.recpack(trk)
	chk(trk, pk)
	k.pk.append(pk)
}

func (k *Packer) recpack(trk Track) Packed {
	n := len(trk)
	pk := k.bitspack(trk)
	if n > 8 {
		n /= 2
		t1, t2 := trk[:n], trk[n:]
		pk1 := k.recpack(t1)
		pk2 := k.recpack(t2)
		if pk1.memsize()+pk2.memsize() < pk.memsize() {
			return pk1.append(pk2)
		}
	}
	return pk
}

func (k *Packer) bitspack(trk Track) Packed {
	var best *Packed
	for _, bp := range k.bits {
		pk := &Packed{acc: k.pk.acc}
		//pk := k.popPacked()
		pk.packpack(trk, bp)
		if best == nil || pk.memsize() < best.memsize() {
			best = pk
		}
	}
	return *best
}

func (k *Packer) popPacked() *Packed {
	n := len(k.stack)
	if n > 0 {
		p := k.stack[n-1]
		k.stack = k.stack[:n-1]
		p.frame = p.frame[:0]
		p.pack = p.pack[:0]
		return p
	}
	p := &Packed{
		acc: k.pk.acc,
	}
	k.buf = append(k.buf, p)
	return p
}

func (k *Packer) pushPacked(p *Packed) {
	if p != nil {
		k.stack = append(k.stack, p)
	}
}

type Packed struct {
	frame []relFrame
	pack  []byte

	acc ptAcc // accuracy reducer
}

func (k Packed) Unpack(dst Track) Track {
	k.ForEach(func(pt Point) error {
		dst = append(dst, pt)
		return nil
	})
	return dst
}

func (k Packed) ForEach(fn func(pt Point) error) error {
	for i, f := range k.frame {
		if err := fn(k.acc.dec(f.pt)); err != nil {
			return err
		}

		j := i + 1
		var pack []byte
		if j < len(k.frame) {
			x := k.frame[j]
			pack = k.pack[f.ofs():x.ofs()]
		} else {
			pack = k.pack[f.ofs():]
		}

		nbytes, dbits := f.elemBytes(), f.dbits()

		for len(pack) > 0 {
			dt, dlat, dlong := relFromBytes(pack[:nbytes], dbits)
			pack = pack[nbytes:]

			pt := Point{
				f.pt.t + dt,
				f.pt.lat + dlat,
				f.pt.long + dlong,
			}
			if err := fn(k.acc.dec(pt)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Packed) At(t time.Time) (lat, long float64) {
	tt := (itime(t) + k.acc.tacc/2) / k.acc.tacc
	i := sort.Search(len(k.frame), func(i int) bool {
		return tt < k.frame[i].pt.t
	})

	if i == 0 {
		if len(k.frame) == 0 {
			return 0, 0
		}
		p := k.acc.dec(k.frame[0].pt)
		return p.Lat(), p.Long()
	}

	f := k.frame[i-1]
	nbytes, dbits := f.elemBytes(), f.dbits()

	var pack []byte
	if i < len(k.frame) {
		x := k.frame[i]
		pack = k.pack[f.ofs():x.ofs()]
	} else {
		pack = k.pack[f.ofs():]
	}

	nelems := len(pack) / nbytes

	ttx := tt - f.pt.t
	j := sort.Search(nelems, func(i int) bool {
		o := i * nbytes
		dt := tFromBytes(pack[o:o+nbytes], dbits)
		return ttx < dt
	})

	var p, q Point
	if j == 0 {
		p = f.pt
		if nelems != 0 {
			dt, dlat, dlong := relFromBytes(pack[:nbytes], dbits)
			q = Point{
				p.t + dt,
				p.lat + dlat,
				p.long + dlong,
			}
		} else if i < len(k.frame) {
			x := k.frame[i]
			q = x.pt
		} else {
			p = k.acc.dec(p)
			return p.Lat(), p.Long()
		}
	} else {
		o := (j - 1) * nbytes
		dt, dlat, dlong := relFromBytes(pack[o:o+nbytes], dbits)
		p = Point{
			f.pt.t + dt,
			f.pt.lat + dlat,
			f.pt.long + dlong,
		}
		if j < nelems {
			o += nbytes
			dt, dlat, dlong := relFromBytes(pack[o:o+nbytes], dbits)
			q = Point{
				f.pt.t + dt,
				f.pt.lat + dlat,
				f.pt.long + dlong,
			}
		} else {
			if i < len(k.frame) {
				q = k.frame[i].pt
			} else {
				// time after last pt
				p = k.acc.dec(p)
				return p.Lat(), p.Long()
			}
		}
	}

	p = k.acc.dec(p)
	q = k.acc.dec(q)

	return trackmath.Interpolate(t,
		p.Time(), p.Lat(), p.Long(),
		q.Time(), q.Lat(), q.Long())
}

func (k *Packed) end() Point {
	n := len(k.frame)
	if n == 0 {
		return Point{}
	}
	f := k.frame[n-1]

	nbytes, dbits := f.elemBytes(), f.dbits()
	last := k.pack[len(k.pack)-nbytes:]

	dt, dlat, dlong := relFromBytes(last, dbits)

	pt := Point{
		f.pt.t + dt,
		f.pt.lat + dlat,
		f.pt.long + dlong,
	}
	return pt
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

func (k *Packed) numpoints() int {
	var n int
	for i, f := range k.frame {
		j := i + 1
		var pk int
		if j < len(k.frame) {
			x := k.frame[j]
			pk = x.ofs() - f.ofs()
		} else {
			pk = len(k.pack) - f.ofs()
		}
		n += 1 + pk/f.elemBytes()
	}
	return n
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

	pkr := NewPacker()
	for _, p := range trk {
		pkr.Add(p)
	}
	pk := pkr.Packed()
	cl := new(countLen)
	n, _ := pk.WriteTo(cl)
	fmt.Println(len(pk.frame), n, cl.n)

	fmt.Println()
}

type countLen struct{ n int64 }

func (c *countLen) Write(p []byte) (n int, err error) {
	c.n += int64(len(p))
	return len(p), nil
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
	for i := 3; i <= 8; i++ {
		s := relPack(trk, i)
		if i == 3 || s.nbytes < best.nbytes {
			best = s
		}
	}
	return best
}

func relPack(trk Track, nbytes int) relPackStat {
	pkr := newBitsPacker(nbytes)

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
	packBytes := uintptr(npk * nbytes)
	return relPackStat{
		bytesPerPack: nbytes,
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
	var lastFrame *relFrame
	for i, p0 := range trk {
		p := pk.acc.enc(p0)
		frame := true
		if i != 0 && bpk.add(p) {
			frame = false
		}

		if frame {
			if lastFrame != nil {
				var dbits uint
				pk.pack, dbits = bpk.pack(pk.pack)
				lastFrame.setDbits(dbits)
			}

			bpk.frame(p)
			pk.frame = append(pk.frame, relFr(p, len(pk.pack), bpk.nbytes, 0))
			lastFrame = &pk.frame[len(pk.frame)-1]
		}
	}

	if !bpk.empty() {
		var dbits uint
		pk.pack, dbits = bpk.pack(pk.pack)
		lastFrame.setDbits(dbits)
	}

	//chk(trk, *pk)
}

func chk(trk Track, pk Packed) {
	var i int
	pk.ForEach(func(p Point) error {
		q := trk[i]
		i++
		dt := p.t - q.t
		dlat := p.lat - q.lat
		dlong := p.long - q.long
		if dt > 50 || dlat > 5 || dlong > 5 {
			fmt.Println(i, dt, dlat, dlong)
			panic("pointerr")
		}
		return nil
	})

	if len(trk) != i {
		panic("hopp")
	}
}

type bitsPacker struct {
	nbytes int

	fr Point

	pt []relPoint

	elem []relPackElem

	idx int // last valid elem idx
}

type relPackElem struct {
	tbits uint
	dbits uint
	valid bool
}

type relPoint struct {
	t         int64
	lat, long uint32
}

func tFromBytes(p []byte, dbits uint) int64 {
	var v uint64
	for _, b := range p {
		v = (v << 8) | uint64(b)
	}

	return int64(v >> (2 * dbits))
}

func relFromBytes(p []byte, dbits uint) (t int64, lat, long int32) {
	var v uint64
	for _, b := range p {
		v = (v << 8) | uint64(b)
	}

	m := uint32(1<<dbits - 1)

	t = int64(v >> (2 * dbits))
	lat = dreldec(uint32(v>>dbits) & m)
	long = dreldec(uint32(v) & m)

	return
}

func (r relPoint) append(dst []byte, nbytes int, dbits uint) []byte {
	v := ((uint64(r.t)<<dbits)|uint64(r.lat))<<dbits | uint64(r.long)

	for n := uint(nbytes); n > 0; {
		n--
		shift := n * 8
		dst = append(dst, byte((v>>shift)&0xff))
	}

	return dst
}

func newBitsPacker(nbytes int) *bitsPacker {
	p := &bitsPacker{
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

func (k *bitsPacker) empty() bool {
	return len(k.pt) == 0
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

	idx := -1
	for i := range k.elem {
		e := &k.elem[i]
		if e.valid {
			if bt <= e.tbits && bd <= e.dbits {
				if idx == -1 {
					idx = i
				}
			} else {
				e.valid = false
			}
		}
	}

	hasvalid := idx >= 0
	if hasvalid {
		k.idx = idx
		k.pt = append(k.pt, relPoint{dt, dlat, dlong})
	}

	return hasvalid
}

func (k *bitsPacker) pack(dst []byte) ([]byte, uint) {
	if k.idx < 0 {
		panic("bitsPacker: illegal pack")
	}
	e := k.elem[k.idx]
	for _, d := range k.pt {
		dst = d.append(dst, k.nbytes, e.dbits)
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
