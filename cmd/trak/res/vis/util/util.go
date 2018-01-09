package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/tajtiattila/track/internal/trackmath"
)

type Visualizer struct {
	*js.Object

	xmap      *js.Object // google.map object
	trackList *js.Object // track list DOM element

	acc float64 // max accuracy value

	mapTrk *js.Object // track polyline on map

	trk []Point // loaded track

	hoverPt trackPointMarker
	selPt   trackPointMarker
}

// point represents a track point
type Point struct {
	Time time.Time `json:"timestamp"`
	Lat  float64   `json:"lat"`
	Long float64   `json:"lon"`
	Acc  float64   `json:"acc"`
}

func NewVisualizer(m, trackList *js.Object) *js.Object {
	vis := &Visualizer{
		xmap:      m,
		trackList: trackList,
		acc:       200,
	}

	vis.hoverPt = trkPtMarker(vis, js.M{
		"strokeColor":   "#000000",
		"strokeOpacity": 0.8,
		"strokeWeight":  1,
		"fillColor":     "#000000",
		"fillOpacity":   0.1,
	})

	vis.selPt = trkPtMarker(vis, js.M{
		"strokeColor":   "#FF0000",
		"strokeOpacity": 0.8,
		"strokeWeight":  2,
		"fillColor":     "#FF0000",
		"fillOpacity":   0.2,
	})
	vis.selPt.addKeyHandler("k", "j", "selected")

	h := js.Global.Get("location").Get("hash").String()
	if h != "" && h[0] == '#' {
		v := strings.SplitN(h[1:], "/", 3)
		if len(v) > 1 {
			if v, err := strconv.ParseFloat(v[1], 64); err == nil {
				vis.acc = v
			}
		}
		vis.Load(v[0])
	}

	trackList.Set("onmousemove", vis.trackHover)
	trackList.Set("onmouseout", vis.trackHoverStop)
	trackList.Set("onclick", vis.trackClick)

	return js.MakeWrapper(vis)
}

func (vis *Visualizer) Load(date string) {
	go func() {
		err := vis.load(date)
		if err != nil {
			println(err)
		}
	}()
}

func (vis *Visualizer) load(date string) error {
	resp, err := http.Get("/api/track/" + date + ".json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var td struct {
		L []Point `json:"locations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&td); err != nil {
		return err
	}

	if len(td.L) == 0 {
		return fmt.Errorf("got empty track")
	}

	vis.trk = td.L
	if vis.mapTrk != nil {
		vis.mapTrk.Call("setMap", nil)
	}

	doc := Doc()

	// bounds
	var b struct {
		la0 float64
		lo0 float64
		la1 float64
		lo1 float64
	}

	var last struct {
		valid bool
		p     Point
		p3    trackmath.Point3
	}

	var trkpt []js.M
	for i, p := range vis.trk {
		if i == 0 {
			b.la0, b.la1 = p.Lat, p.Lat
			b.lo0, b.lo1 = p.Long, p.Long
		} else {
			switch {
			case p.Lat < b.la0:
				b.la0 = p.Lat
			case p.Lat > b.la1:
				b.la1 = p.Lat
			}
			switch {
			case p.Long < b.lo0:
				b.lo0 = p.Long
			case p.Long > b.lo1:
				b.lo1 = p.Long
			}
		}

		usept := p.Acc <= vis.acc

		var clsex string
		if !usept {
			clsex = " inaccurate"
		}
		div := doc.CreateElement("div", js.M{
			"id":        fmt.Sprintf("trkpt%d", i),
			"className": "trkpt" + clsex,
		})
		vis.trackList.Call("appendChild", div)

		ts := doc.CreateElement("span", js.M{
			"className": "timestamp",
		}, p.Time.Local().Format("15:04:05.000"))
		div.Call("appendChild", ts)

		div.Call("appendChild", doc.CreateElement("span", js.M{
			"className": "accuracy",
		}, fmt.Sprint(p.Acc)))

		if usept {
			trkpt = append(trkpt, js.M{
				"lat": p.Lat,
				"lng": p.Long,
			})

			p3 := trackmath.Pt3(p.Lat, p.Long)
			if last.valid {
				dv := p3.Sub(last.p3)
				dt := p.Time.Sub(last.p.Time)

				div.Call("appendChild", doc.CreateElement("span", js.M{
					"className": "dist",
				}, fmt.Sprintf("%.0f", dv.Mag())))

				div.Call("appendChild", doc.CreateElement("span", js.M{
					"className": "timedelta",
				}, fmt.Sprintf("%7s", durationStr(dt))))
			}

			last.valid = true
			last.p = p
			last.p3 = p3
		}
	}

	vis.xmap.Call("fitBounds", js.M{
		"north": b.la1,
		"south": b.la0,
		"west":  b.lo0,
		"east":  b.lo1,
	})

	vis.mapTrk = NewMapObj("Polyline", js.M{
		"path": trkpt,

		"strokeColor":   "#0000FF",
		"strokeOpacity": 0.4,
		"strokeWidth":   2,

		"map": vis.xmap,
	})

	return nil
}

func (vis *Visualizer) trackHover(evt *js.Object) {
	div, idx := getTrkptDiv(evt.Get("target"))
	if div != nil {
		vis.hoverPt.Show(idx, false)
	}
}

func (vis *Visualizer) trackHoverStop(evt *js.Object) {
	vis.hoverPt.Hide()
}

func (vis *Visualizer) trackClick(evt *js.Object) {
	div, idx := getTrkptDiv(evt.Get("target"))
	if div != nil {
		vis.selPt.Show(idx, true)
	}
}

type rect struct {
	north, east, south, west float64
}

func (r rect) pointInside(lat, long float64) bool {
	if lat < r.south || lat > r.north {
		return false
	}

	if r.east < r.west {
		r.east += 360
		long += 360
	}

	return r.west <= long && long <= r.east
}

func (vis *Visualizer) mapBounds() rect {
	r := vis.xmap.Call("getBounds")
	ne := r.Call("getNorthEast")
	sw := r.Call("getSouthWest")
	return rect{
		north: ne.Call("lat").Float(),
		east:  ne.Call("lng").Float(),
		south: sw.Call("lat").Float(),
		west:  sw.Call("lng").Float(),
	}
}

func getTrkptDiv(node *js.Object) (div *js.Object, idx int) {
	for node != nil && !strings.HasPrefix(node.Get("id").String(), "trkpt") {
		node = node.Get("parentNode")
	}
	if node == nil {
		return nil, -1
	}
	idx, err := strconv.Atoi(strings.TrimPrefix(node.Get("id").String(), "trkpt"))
	if err != nil {
		return nil, -1
	}
	return node, idx
}

func main() {
	js.Global.Set("newVisualizer", NewVisualizer)
}

type Document struct {
	*js.Object
}

func Doc() *Document {
	return &Document{
		Object: js.Global.Get("document"),
	}
}

func (d *Document) CreateElement(name string, attr js.M, content ...string) *js.Object {
	e := d.Call("createElement", name)
	for k, v := range attr {
		e.Set(k, v)
	}
	if len(content) != 0 {
		e.Set("innerHTML", html.EscapeString(strings.Join(content, "")))
	}
	return e
}

func (d *Document) GetElementByID(id string) *js.Object {
	return d.Call("getElementById", id)
}

func NewMapObj(name string, attrs js.M) *js.Object {
	cls := js.Global.Get("google").Get("maps").Get(name)
	return cls.New(attrs)
}

func durationStr(d time.Duration) string {
	if d < 0 {
		return d.String()
	}
	if d < time.Minute {
		if d < 9*time.Second {
			return fmt.Sprintf("%.1fs", float64(d)/float64(time.Second))
		}
		return d.Truncate(time.Second).String()
	}
	return strings.TrimRight(d.Truncate(time.Minute).String(), "0s")
}
