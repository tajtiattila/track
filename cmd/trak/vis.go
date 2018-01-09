package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tajtiattila/basedir"
	"github.com/tajtiattila/cmdmain"
	"github.com/tajtiattila/track/trackio"
)

type VisualizeCmd struct {
	listen   string
	res      string
	gopherjs bool

	trk   map[string]trackio.Track
	dates []string // local dates with track; 2006-01-02
}

func init() {
	cmdmain.Register("vis", func(flags *flag.FlagSet) cmdmain.Command {
		c := new(VisualizeCmd)
		flags.StringVar(&c.listen, "listen", ":8475", "listen address")
		flags.StringVar(&c.res, "res", "", "resource (html, css...) directory")
		flags.BoolVar(&c.gopherjs, "gopherjs", false, "use gopherjs serve to serve util.js")
		return c
	})
}

func (*VisualizeCmd) Describe() string {
	return "Visualize track."
}

func (*VisualizeCmd) ArgNames() string {
	return "[track]"
}

func (c *VisualizeCmd) Run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("need one path argument")
	}

	const resdir = "github.com/tajtiattila/track/cmd/trak/res/vis"

	if c.res == "" {
		var err error
		c.res, err = basedir.Gopath.Dir("src/" + resdir)
		if err != nil {
			return err
		}
	}

	xch := make(chan error)
	go func() {
		if c.gopherjs {
			xch <- serveGopherJS("/"+resdir, "util")
		} else {
			xch <- nil
		}
	}()

	trk, err := loadRaw(args[0])
	if err != nil {
		return err
	}

	c.trk = make(map[string]trackio.Track)
	for _, p := range trk {
		ds := p.Time.Local().Format("2006-01-02")
		v := c.trk[ds]
		v = append(v, p)
		c.trk[ds] = v
	}

	for k := range c.trk {
		c.dates = append(c.dates, k)
	}
	sort.Strings(c.dates)

	if len(c.dates) == 0 {
		return fmt.Errorf("no tracks to serve")
	}

	if err := <-xch; err != nil {
		return err
	}

	return c.serve()
}

func (c *VisualizeCmd) serve() error {
	const varname = "GOOGLEMAPS_APIKEY"
	apikey := os.Getenv(varname)
	if apikey == "" {
		return fmt.Errorf("%v env var missing", varname)
	}

	td := struct {
		GoogleMapsAPIKey string
	}{
		GoogleMapsAPIKey: apikey,
	}

	http.Handle("/", http.FileServer(&templateDir{c.res, td}))
	http.HandleFunc("/api/appdata.json", c.ServeAppData)
	handleWithPrefix("/api/track/", http.HandlerFunc(c.ServeTrack))

	go func() {
		time.Sleep(time.Second)
		host, port, err := net.SplitHostPort(c.listen)
		if err != nil {
			fmt.Println(err)
			return
		}
		if host == "" {
			host = "localhost"
		}
		err = openbrowser(fmt.Sprintf("http://%s:%s", host, port))
		if err != nil {
			fmt.Println(err)
		}
	}()

	fmt.Println("listening on", c.listen)
	return http.ListenAndServe(c.listen, nil)
}

func serveGopherJS(srcRoot string, module ...string) error {
	if len(srcRoot) == 0 || srcRoot[0] != '/' {
		panic("serveGopherJS srcRoot invalid")
	}

	const port = ":8474"
	cmd := exec.Command("gopherjs", "serve", "--http="+port)

	// set gopherjs GOOS to darwin, see
	// https://github.com/gopherjs/gopherjs/issues/688
	if runtime.GOOS == "windows" {
		cmd.Env = append(os.Environ(), "GOOS=darwin")
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	u, err := url.Parse("http://localhost" + port + "/")
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(u)

	// handle for sources
	http.Handle(srcRoot, proxy)

	// modules
	for _, m := range module {
		pfx := srcRoot + "/" + m
		http.Handle("/"+m+".js", addPrefix(pfx, proxy))
		http.Handle("/"+m+".js.map", addPrefix(pfx, proxy))
	}

	uu, err := u.Parse("/util.js")
	if err != nil {
		return err
	}

	return tryURL(uu, 30*time.Second)
}

func addPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = prefix + r.URL.Path
		h.ServeHTTP(w, r2)
	})
}

const noAccuracy = 1e9

func (c *VisualizeCmd) ServeAppData(w http.ResponseWriter, req *http.Request) {
	type AppData struct {
		Dates []string `json:"dates"`

		Start string      `json:"startDate"`
		Trk   interface{} `json:"track"`
	}

	last := c.dates[len(c.dates)-1]
	trk := c.trk[last]

	d := AppData{
		Dates: c.dates,
		Start: last,
		Trk:   trackData(trk, noAccuracy),
	}

	err := json.NewEncoder(w).Encode(d)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *VisualizeCmd) ServeTrack(w http.ResponseWriter, req *http.Request) {
	k := strings.TrimRight(req.URL.Path, ".json")
	if strings.HasPrefix(k, "/") {
		k = k[1:]
	}
	v := req.URL.Query()
	acc := noAccuracy
	if n, err := strconv.ParseFloat(v.Get("accuracy"), 64); err == nil {
		acc = n
	}

	trk := c.trk[k]
	if len(trk) == 0 {
		http.NotFound(w, req)
		return
	}

	err := json.NewEncoder(w).Encode(trackData(trk, acc))
	if err != nil {
		fmt.Println(err)
	}
}

func trackData(trk trackio.Track, acc float64) interface{} {
	type Jpt struct {
		Ts  string  `json:"timestamp"`
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
		Acc float64 `json:"acc"`
	}
	type Bounds struct {
		Lat0 float64 `json:"lat0"`
		Lon0 float64 `json:"lon0"`
		Lat1 float64 `json:"lat1"`
		Lon1 float64 `json:"lon1"`
	}
	type TrkData struct {
		B Bounds `json:"bounds"`
		L []Jpt  `json:"locations"`
	}

	var d TrkData
	for _, p := range trk {
		if p.Acc > acc {
			continue
		}

		if len(d.L) == 0 {
			d.B.Lat0, d.B.Lat1 = p.Lat, p.Lat
			d.B.Lon0, d.B.Lon1 = p.Long, p.Long
		} else {
			switch {
			case p.Lat < d.B.Lat0:
				d.B.Lat0 = p.Lat
			case p.Lat > d.B.Lat1:
				d.B.Lat1 = p.Lat
			}
			switch {
			case p.Long < d.B.Lon0:
				d.B.Lon0 = p.Long
			case p.Long > d.B.Lon1:
				d.B.Lon1 = p.Long
			}
		}
		jp := Jpt{
			Ts:  p.Time.UTC().Format("2006-01-02T15:04:05.999Z"),
			Lat: p.Lat,
			Lon: p.Long,
			Acc: p.Acc,
		}
		d.L = append(d.L, jp)
	}

	return d
}

func withCwd(path string, f func() error) (err error) {
	var old string
	if old, err = os.Getwd(); err != nil {
		return err
	}
	err = os.Chdir(path)
	if err != nil {
		return err
	}

	defer func() {
		xerr := os.Chdir(old)
		if err == nil {
			err = xerr
		}
	}()

	return f()
}

// templateDir is like http.Dir but applies
// the template arguments to html files.
type templateDir struct {
	root string
	data interface{} // template data
}

func (td *templateDir) Open(name string) (http.File, error) {
	f, err := http.Dir(td.root).Open(name)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(name, ".html") {
		return f, nil
	}

	defer f.Close()

	src, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	t, err := template.New(name).Parse(string(src))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, td.data); err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return newFakeFile(fi, bytes.NewReader(buf.Bytes())), nil
}

type fakeFile struct {
	name    string
	modTime time.Time

	r *bytes.Reader
}

func newFakeFile(fi os.FileInfo, r *bytes.Reader) *fakeFile {
	return &fakeFile{
		name:    fi.Name(),
		modTime: fi.ModTime(),
		r:       r,
	}
}

func (f *fakeFile) Close() error                                 { return nil }
func (f *fakeFile) Read(p []byte) (n int, err error)             { return f.r.Read(p) }
func (f *fakeFile) Seek(offset int64, whence int) (int64, error) { return f.r.Seek(offset, whence) }
func (f *fakeFile) Readdir(count int) ([]os.FileInfo, error)     { return nil, os.ErrInvalid }
func (f *fakeFile) Stat() (os.FileInfo, error)                   { return f, nil }

// fakeFile as os.FileInfo
func (f *fakeFile) Name() string       { return f.name }
func (f *fakeFile) Size() int64        { return f.r.Size() }
func (f *fakeFile) Mode() os.FileMode  { return 0666 }
func (f *fakeFile) ModTime() time.Time { return f.modTime }
func (f *fakeFile) IsDir() bool        { return false }
func (f *fakeFile) Sys() interface{}   { return nil }

func handleWithPrefix(pfx string, h http.Handler) {
	n := len(pfx) - 1
	if n < 0 || pfx[n] != '/' {
		panic("prefix must end in '/'")
	}
	http.Handle(pfx, http.StripPrefix(pfx[:n], h))
}

// https://gist.github.com/hyg/9c4afcd91fe24316cbf0
func openbrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func tryURL(u *url.URL, timeout time.Duration) error {
	start := time.Now()
	for {
		_, err := http.Get(u.String())
		if err == nil {
			return err
		}
		elapsed := time.Now().Sub(start)
		if elapsed > timeout {
			return err
		}
		time.Sleep(time.Second)
	}
	panic("unreachable")
}
