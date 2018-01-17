package track

import "unsafe"

func (trk Track) Memsize() uintptr {
	return unsafe.Sizeof(trk) + uintptr(len(trk))*unsafe.Sizeof(Point{})
}

func (k Packed) Memsize() uintptr {
	return unsafe.Sizeof(k) + k.memsize()
}
