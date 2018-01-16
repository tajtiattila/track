package testutil

import (
	"math/rand"
	"testing"
	"time"

	"github.com/tajtiattila/track"
)

// TrackTimes returns time stamps between StartTime and EndTime of trk
func TrackTimes(trk track.Track) <-chan time.Time {
	ch := make(chan time.Time)
	st, et := trk.StartTime(), trk.EndTime()
	go func() {
		defer close(ch)

		for i := -10; i <= 10; i++ {
			dt := time.Duration(i) * time.Second
			ch <- st.Add(dt)
			ch <- et.Add(dt)
		}

		if !testing.Short() {
			// whole seconds
			stt := st.Truncate(time.Second)
			ett := et.Truncate(time.Second).Add(time.Second)
			for t := stt; !t.After(ett); t = t.Add(time.Second) {
				ch <- t
			}
		}

		// yield pseudo random but deterministic times
		rnd := rand.New(rand.NewSource(st.Unix()))
		xt := int64(et.Sub(st))
		if xt != 0 {
			for i := 0; i < 100; i++ {
				dt := time.Duration(rnd.Int63n(xt))
				ch <- st.Add(dt)
			}
		}
	}()
	return ch
}
