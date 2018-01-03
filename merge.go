package track

import (
	"sort"
)

// Merge updates *ptrk by replacing its track points
// between seg.StartTime() and seg.EndTime() with
// track points from seg.
func (ptrk *Track) Merge(seg Track) {
	if len(seg) == 0 {
		return
	}

	if len(*ptrk) == 0 {
		*ptrk = append(*ptrk, seg...)
		return
	}

	trk := *ptrk

	ss := seg[0].t
	se := seg[len(seg)-1].t

	si := sort.Search(len(trk), func(i int) bool {
		return ss <= trk[i].t
	})

	ei := sort.Search(len(trk), func(i int) bool {
		return se < trk[i].t
	})

	var tt Track
	o := si + len(seg)
	n := o + len(trk[ei:])
	if cap(trk) < n {
		tt = make(Track, n)
		copy(tt, trk[:si])
		copy(tt[si:], seg)
		copy(tt[o:], trk[ei:])
		trk = tt
	} else {
		tt = trk[:n]
		copy(tt[o:], trk[ei:])
		copy(tt[si:], seg)
	}

	*ptrk = tt
}
