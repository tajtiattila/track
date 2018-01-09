package trackio_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/tajtiattila/track/trackio"
)

var sampleKML = `<?xml version='1.0' encoding='UTF-8'?>
<kml xmlns='http://www.opengis.net/kml/2.2' xmlns:gx='http://www.google.com/kml/ext/2.2'>
	<Document>
		<Placemark>
			<open>1</open>
			<gx:Track>
				<altitudeMode>clampToGround</altitudeMode>
				<when>2011-04-26T07:54:05Z</when>
				<gx:coord>54.7805978 9.4376219 100</gx:coord>
				<when>2011-04-26T07:55:45Z</when>
				<gx:coord>54.7793527 9.4382112 0</gx:coord>
				<when>2011-04-26T07:57:22Z</when>
				<gx:coord>54.7780257 9.4383682 0</gx:coord>
				<when>2011-04-26T07:59:59Z</when>
				<gx:coord>54.7764576 9.4373156 0</gx:coord>
				<when>2011-04-26T08:01:44Z</when>
				<when>2011-04-26T08:02:17Z</when>
				<when>2011-04-26T08:06:26Z</when>
				<when>2011-04-26T08:06:35Z</when>
				<when>2011-04-26T08:06:42Z</when>
				<when>2011-04-26T08:07:00Z</when>
				<when>2011-04-26T08:07:12Z</when>
				<when>2011-04-26T08:07:34Z</when>
				<when>2011-04-26T08:09:08Z</when>
				<when>2011-04-26T08:09:35Z</when>
				<when>2011-04-26T08:09:42Z</when>
				<when>2011-04-26T08:11:32Z</when>
				<when>2011-04-26T08:12:51Z</when>
				<when>2011-04-26T08:13:32Z</when>
				<when>2011-04-26T08:14:34Z</when>
				<when>2011-04-26T08:18:11Z</when>
				<when>2011-04-26T08:19:23Z</when>
				<when>2011-04-26T08:21:19Z</when>
				<when>2011-04-26T08:23:13Z</when>
				<when>2011-04-26T08:24:25Z</when>
				<when>2011-04-26T08:25:51Z</when>
				<gx:coord>54.7751258 9.4364726 156</gx:coord>
				<gx:coord>54.7750663 9.4359833 156</gx:coord>
				<gx:coord>54.7749039 9.4356500 156</gx:coord>
				<gx:coord>54.7748850 9.4355465 156</gx:coord>
				<gx:coord>54.7749339 9.4355597 156</gx:coord>
				<gx:coord>54.7750490 9.4355169 156</gx:coord>
				<gx:coord>54.7750379 9.4353671 156</gx:coord>
				<gx:coord>54.7748416 9.4353480 156</gx:coord>
				<gx:coord>54.7737402 9.4354813 156</gx:coord>
				<gx:coord>54.7735581 9.4357957 156</gx:coord>
				<when>2011-04-26T08:14:42Z</when>
				<when>2011-04-26T08:15:08Z</when>
				<when>2011-04-26T08:15:26Z</when>
				<when>2011-04-26T08:16:34Z</when>
				<gx:coord>54.7735632 9.4358950 156</gx:coord>
				<gx:coord>54.7731582 9.4380101 156</gx:coord>
				<gx:coord>54.7726949 9.4391077 156</gx:coord>
				<gx:coord>54.7725256 9.4393897 156</gx:coord>
				<gx:coord>54.7719606 9.4400140 156</gx:coord>
				<gx:coord>54.7719509 9.4400266 156</gx:coord>
				<gx:coord>54.7718998 9.4400669 156</gx:coord>
				<gx:coord>54.7718369 9.4401513 156</gx:coord>
				<gx:coord>54.7712847 9.4412500 156</gx:coord>
				<gx:coord>54.7714076 9.4429594 156</gx:coord>
				<gx:coord>54.7709788 9.4444952 156</gx:coord>
				<gx:coord>54.7696980 9.4451720 156</gx:coord>
				<gx:coord>54.7683050 9.4458231 156</gx:coord>
				<gx:coord>54.7673059 9.4460718 156</gx:coord>
				<gx:coord>54.7665063 9.4447323 156</gx:coord>
			</gx:Track>
		</Placemark>
	</Document>
</kml>`

func TestKML(t *testing.T) {
	r := bytes.NewReader([]byte(sampleKML))
	trk, err := trackio.NewDecoder(r).Track()
	if err != nil {
		t.Fatal(err)
	}

	const wantLen = 29
	if len(trk) != wantLen {
		t.Fatalf("track length mismatch: want %d got %d", wantLen, len(trk))
	}

	pointEqual(t, trk[0], trackio.Pt(
		time.Date(2011, 4, 26, 7, 54, 5, 0, time.UTC),
		54.7805978,
		9.4376219,
	))
}

var sampleKMLLengthMismatch = `<?xml version='1.0' encoding='UTF-8'?>
<kml xmlns='http://www.opengis.net/kml/2.2' xmlns:gx='http://www.google.com/kml/ext/2.2'>
	<Document>
		<Placemark>
			<open>1</open>
			<gx:Track>
				<altitudeMode>clampToGround</altitudeMode>
				<when>2011-04-26T07:54:05Z</when>
				<gx:coord>54.7805978 9.4376219 100</gx:coord>
				<when>2011-04-26T07:55:45Z</when>
				<gx:coord>54.7793527 9.4382112 0</gx:coord>
				<when>2011-04-26T07:57:22Z</when>
				<gx:coord>54.7780257 9.4383682 0</gx:coord>
				<when>2011-04-26T07:59:59Z</when>
				<gx:coord>54.7764576 9.4373156 0</gx:coord>
				<when>2011-04-26T08:01:44Z</when>
				<when>2011-04-26T08:02:17Z</when>
				<when>2011-04-26T08:06:26Z</when>
				<when>2011-04-26T08:06:35Z</when>
				<when>2011-04-26T08:06:42Z</when>
				<when>2011-04-26T08:07:00Z</when>
				<when>2011-04-26T08:07:12Z</when>
				<when>2011-04-26T08:07:34Z</when>
				<when>2011-04-26T08:09:08Z</when>
				<when>2011-04-26T08:09:35Z</when>
			</gx:Track>
		</Placemark>
	</Document>
</kml>`

func TestKMLLengthMismatch(t *testing.T) {
	r := bytes.NewReader([]byte(sampleKMLLengthMismatch))
	_, err := trackio.NewDecoder(r).Track()
	if err == nil || strings.Index(err.Error(), "length mismatch") == -1 {
		t.Fatalf("got %v, want length mismatch error", err)
	}
}
