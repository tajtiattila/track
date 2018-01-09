package trackio

import "time"

// NoAccuracy is used in place of the accuracy value
// if the actual value is not known.
//
// For instance KML has no accuracy information.
const NoAccuracy = 1e9

// Point is a GPS track point.
type Point struct {
	Time time.Time // time stamp of track point

	Lat  float64 // degrees of latitude
	Long float64 // degrees of longitude

	Acc float64 // estimated horizontal accuracy (meters)

	Ele Elevation // elevation/altitude information
}

func Pt(t time.Time, lat, long float64) Point {
	return Point{
		Time: t,
		Lat:  lat,
		Long: long,
		Acc:  NoAccuracy,
		Ele:  Elevation{Acc: NoAccuracy},
	}
}

// Elevation holds GPS track point elevation/altitude information.
type Elevation struct {
	Valid   bool    // indicates if Float64 is and Acc may be valid (always true for KML)
	Float64 float64 // elevation value in meters above sea level
	Acc     float64 // estimated vertical accuracy (meters)
}
