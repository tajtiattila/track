package tracksimpl

import (
	"container/heap"

	"github.com/tajtiattila/track"
)

// EndPointFit implements a variant of
// the Ramer–Douglas–Peucker iterative end-point fit algorithm,
// also known as the Douglas–Peucker algorithm
// that provides an excellent approximation of the original track.
//
// It has an expected comlexity of O(n*log(n)).
// If Full is true, it has a worst case complexity of O(n²).
type EndPointFit struct {
	D float64 // maximum error distance in meters

	// If Full is false, the algorithm uses a recursion depth limiter
	// to reduce memory use and improve performance.
	// If Full is true, the recursion depth limiter is turned off.
	Full bool
}

const parallelWin = 64

func (x EndPointFit) Run(dst, src track.Track) track.Track {
	n := len(src)
	if n <= 2 {
		return append(dst, src...)
	}

	last := src[n-1]

	f := epf{
		dd: x.D * x.D,
	}
	if x.Full {
		f.maxDepth = n
	} else {
		f.maxDepth = 1
		for m := 1; m < n; m *= 2 {
			f.maxDepth++
		}
		f.maxDepth *= 2 // arbitrary
	}

	if n <= parallelWin {
		// avoid gorouting/channel setup
		return append(f.step(0, dst, src), last)
	}

	return append(f.run(0, dst, src), last)
}

type epf struct {
	dd float64

	maxDepth int

	buf track.Track
	ch  chan work
}

// step performs the iterative end-point fit algorithm recursively,
// but does not append the last point of src to dst.
func (f *epf) step(depth int, dst, src track.Track) track.Track {
	i, res := f.findSplit(src)

	if i < 0 {
		return append(dst, res...)
	}

	n := len(src) - 1

	depth++

	const adaptiveWin = 32

	if depth >= f.maxDepth && n > adaptiveWin {
		o := n / 4
		m := n / 2
		if i < o {
			dst = f.step(depth, dst, src[:i+1])
			dst = f.step(depth, dst, src[i:m+1])
			return f.step(depth, dst, src[m:])
		} else if i+o > n {
			dst = f.step(depth, dst, src[:m+1])
			dst = f.step(depth, dst, src[m:i+1])
			return f.step(depth, dst, src[i:])
		}
	}

	dst = f.step(depth, dst, src[:i+1])
	return f.step(depth, dst, src[i:])
}

// findSplit finds the optimal split point i in src.
// If no further split is necessary (simplification is done)
// then it returns i == -1 along with the simplified track.
func (f *epf) findSplit(src track.Track) (i int, simpl track.Track) {
	n := len(src) - 1
	if n < 2 {
		return -1, src[:n]
	}

	a, b := src[0], src[n]
	a3, b3 := pt3(a), pt3(b)

	dt := float64(b.Time().Sub(a.Time())) // nanoseconds
	if dt < 1 {
		return -1, src[:n]
	}
	v := b3.Sub(a3).Muls(1 / dt) // meters/nanosecond

	var imax int
	var dmax float64
	for i := 1; i < n; i++ {
		p := src[i]
		p3 := pt3(p)

		dt = float64(p.Time().Sub(a.Time()))
		q3 := a3.Add(v.Muls(dt))

		if d := dist3sq(p3, q3); d > dmax {
			imax, dmax = i, d
		}
	}

	if dmax <= f.dd {
		return -1, src[:1]
	}
	return imax, nil
}

type work struct {
	// offset and length in src that yielded result
	first, last int

	result track.Track
}

// run runs the iterative end-point fit algorithm
// using goroutines and assembles the result.
func (f *epf) run(depth int, dst, src track.Track) track.Track {
	f.ch = make(chan work, 16)

	n := len(src) - 1

	go func() {
		f.bigStep(depth, src, 0, n)
	}()

	wh := make(workHeap, 0, n*4/parallelWin)
	h := heap.Interface(&wh)

	var ndst int
	parts := make([]track.Track, 0, n*4/parallelWin)

	ofs := 0
	for ofs < n {
		w := <-f.ch
		if w.first == ofs {
			ofs = w.last
			parts = append(parts, w.result)
			ndst += len(w.result)

			for len(wh) > 0 && wh[0].first == ofs {
				w = wh[0]
				heap.Pop(h)

				ofs = w.last
				parts = append(parts, w.result)
				ndst += len(w.result)
			}
		} else {
			heap.Push(h, w)
		}
	}

	if cap(dst)-len(dst) < ndst+1 {
		x := make(track.Track, len(dst), len(dst)+ndst+1)
		copy(x, dst)
		dst = x
	}
	for _, p := range parts {
		dst = append(dst, p...)
	}

	return dst
}

// bigStep runs the iterative end-point fit algorithm
// on src[first:last+1] recursively, putting results in f.ch.
// If executes splits using goroutines.
func (f *epf) bigStep(depth int, src track.Track, first, last int) {
	if last-first < parallelWin || depth > f.maxDepth {
		buf := f.step(depth, nil, src[first:last+1])
		f.ch <- work{
			first:  first,
			last:   last,
			result: buf,
		}
		return
	}

	i, res := f.findSplit(src[first : last+1])

	if i < 0 {
		f.ch <- work{
			first:  first,
			last:   last,
			result: res,
		}
	} else {
		i += first
		depth++

		go f.bigStep(depth, src, first, i)
		f.bigStep(depth, src, i, last)
	}
}

type workHeap []work

func (h *workHeap) Len() int           { return len(*h) }
func (h *workHeap) Less(i, j int) bool { return (*h)[i].first < (*h)[j].first }
func (h *workHeap) Swap(i, j int)      { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }

func (h *workHeap) Push(x interface{}) {
	(*h) = append(*h, x.(work))
}

func (h *workHeap) Pop() interface{} {
	n := len(*h) - 1
	x := (*h)[n]
	*h = (*h)[:n]
	return x
}
