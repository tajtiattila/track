package main

import (
	"fmt"
	"math"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jquery"
)

type trackPointMarker struct {
	vis   *Visualizer
	attrs js.M

	selClass string

	circleObj   *js.Object
	latLineObj  *js.Object
	longLineObj *js.Object

	idx int // current index; only if circleObj != nil
}

func trkPtMarker(vis *Visualizer, attrs js.M) trackPointMarker {
	return trackPointMarker{
		vis:   vis,
		attrs: attrs,
	}
}

func (m *trackPointMarker) addKeyHandler(up, down string, selCls string) {
	m.selClass = selCls

	keydown := func(evt *js.Object) {
		k := evt.Get("key").String()
		if k != up && k != down {
			return
		}

		var sel int
		if m.circleObj == nil {
			sel = 0
		} else if k == up {
			sel = m.idx - 1
		} else {
			sel = m.idx + 1
		}
		if 0 <= sel && sel < len(m.vis.trk) {
			m.Show(sel, false)
		}
	}

	js.Global.Get("document").Call("addEventListener", "keydown", keydown, false)
}

func (m *trackPointMarker) Show(idx int, anim bool) {
	if m.circleObj != nil && idx == m.idx {
		return
	}

	m.idx = idx
	if m.circleObj != nil {
		m.circleObj.Call("setMap", nil)
		m.latLineObj.Call("setMap", nil)
		m.longLineObj.Call("setMap", nil)
	}

	p := m.vis.trk[m.idx]

	cattrs := make(js.M)
	lattrs := make(js.M)
	for k, v := range m.attrs {
		cattrs[k] = v
		lattrs[k] = v
	}

	const minLine = 200
	dist := p.Acc * 1.2
	if dist < minLine {
		dist = minLine
	}
	ns := metersToLat(dist)
	ew := metersToLong(p.Lat, dist)
	n := latLng(p.Lat+ns, p.Long)
	s := latLng(p.Lat-ns, p.Long)
	e := latLng(p.Lat, p.Long+ew)
	w := latLng(p.Lat, p.Long-ew)

	m.circleObj = NewMapObj("Circle", mAppend(m.attrs, js.M{
		"center": latLng(p.Lat, p.Long),
		"radius": p.Acc,
		"map":    m.vis.xmap,
	}))
	m.latLineObj = NewMapObj("Polyline", mAppend(m.attrs, js.M{
		"path": []js.M{n, s},
		"map":  m.vis.xmap,
	}))
	m.longLineObj = NewMapObj("Polyline", mAppend(m.attrs, js.M{
		"path": []js.M{e, w},
		"map":  m.vis.xmap,
	}))

	if m.selClass != "" {
		jq := jquery.NewJQuery
		jq("#track").Children(nil).RemoveClass(m.selClass)

		el := Doc().GetElementByID(fmt.Sprintf("trkpt%d", m.idx))
		jq(el).AddClass(m.selClass)

		scrollIntoView(el, Doc().GetElementByID("sidebar-content"), anim)

		if !m.vis.mapBounds().pointInside(p.Lat, p.Long) {
			m.vis.xmap.Call("setCenter", js.M{
				"lat": p.Lat,
				"lng": p.Long,
			})
		}
	}
}

func (m *trackPointMarker) Hide() {
	if m.selClass != "" {
		jq := jquery.NewJQuery
		jq("#track").Children(nil).RemoveClass(m.selClass)
	}

	if m.circleObj != nil {
		m.circleObj.Call("setMap", nil)
		m.latLineObj.Call("setMap", nil)
		m.longLineObj.Call("setMap", nil)

		m.circleObj = nil
		m.latLineObj = nil
		m.longLineObj = nil
	}
}

func latLng(lat, long float64) js.M {
	return js.M{
		"lat": lat,
		"lng": long,
	}
}

func metersToLat(m float64) float64 {
	return m * 360 / 40e6
}

func metersToLong(lat, m float64) float64 {
	k := 1 / math.Cos(lat*math.Pi/180)
	return k * metersToLat(m)
}

// mAppend appends base attrs missing from m to m, and returns m
func mAppend(base, m js.M) js.M {
	if m == nil {
		m = make(js.M)
	}
	for k, v := range base {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m
}

func scrollIntoView(elem, scroll *js.Object, anim bool) {
	jq := jquery.NewJQuery

	et := jq(elem).Offset().Top
	st := jq(scroll).Offset().Top

	eh := jq(elem).Height()
	sh := jq(scroll).Height()

	lines := int(float64(sh) / float64(eh))
	if lines > 3 {
		lines = 3
	}
	margin := lines * eh

	d := et - margin - st
	if d >= 0 {
		d = et + eh + margin - (st + sh)
		if d <= 0 {
			return
		}
	}

	scrollTo := jq(scroll).ScrollTop() + d
	if anim {
		/*jq(scroll).Animate(js.M{
			"scrollTop": scrollTo,
		}, 100)*/
		jq(scroll).Call("animate", js.M{
			"scrollTop": scrollTo,
		}, 100)
	} else {
		jq(scroll).SetScrollTop(scrollTo)
	}
}
