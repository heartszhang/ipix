package main

import (
	"bytes"
	"math"
	"strconv"
	"strings"
)

type record struct {
	Min      uint32
	Max      uint32
	Addr     uint32
	ALen     uint32
	Owner    uint32
	OLen     uint32
	BDLon    [12]byte
	BDLat    [12]byte
	WGSLon   [12]byte
	WGSLat   [12]byte
	Radius   [12]byte
	Scene    [12]byte
	Accuracy [12]byte
}

type address struct {
	Continents string `json:"c,omitempty"`
	Code       string `json:"code,omitempty"`
	Zip        string `json:"zip,omitempty"`
	Country    string `json:"country,omitempty"`
	Province   string `json:"prov,omitempty"`
	City       string `json:"city,omitempty"`
	District   string `json:"district,omitempty"`
}

func saddr(index uint32, s string) *address {
	if x, ok := config.addrs[index]; ok {
		return x
	}
	secs := strings.Split(s, "|")
	if len(secs) == 7 {
		x := &address{
			secs[0], secs[1], secs[2], secs[3], secs[4], secs[5], secs[6],
		}
		config.addrs[index] = x
		return x
	}
	return nil
}

type item struct {
	*address
	WGS      string `json:"wgs,omitempty"`
	Radius   string `json:"radius,omitempty"`
	Scene    string `json:"scene,omitempty"`
	Accuracy string `json:"accuracy,omitempty"`
	lat      float64
	lng      float64
	min      uint32
	max      uint32
}

func toitem(r *record, dict []byte) *item {
	return &item{
		address:  saddr(r.Addr, string(dict[r.Addr:r.Addr+r.ALen])),
		WGS:      btrim(r.WGSLat) + "," + btrim(r.WGSLon),
		Accuracy: btrim(r.Accuracy),
		Radius:   btrim(r.Radius),
		Scene:    btrim(r.Scene),
		max:      r.Max,
		min:      r.Min,
		lat:      a2f(btrim(r.WGSLat)),
		lng:      a2f(btrim(r.WGSLon)),
	}
}

func a2f(str string) float64 {
	f, _ := strconv.ParseFloat(str, 64)
	return f
}
func btrim(i [12]byte) string {
	return string(bytes.Trim(i[:], null))
}

const er = 6371.0 //km

func r(x float64) float64 {
	return x * math.Pi / 180
}

// for small distances Pythagoras’ theorem can be used on an equi­rectangular projec­tion
func equirectangular(a, b *item) float64 {
	var cos, sqrt = math.Cos, math.Sqrt
	x := r(a.lng-b.lng) * cos(r(a.lat+b.lat)/2)
	y := r(a.lat - b.lat)
	d := sqrt(x*x+y*y) * er
	return d
}

//uses the ‘haversine’ formula to calculate the great-circle haversine between two points
func haversine(a, b *item) float64 {
	var cos, atan2, sqrt, sin = math.Cos, math.Atan2, math.Sqrt, math.Sin
	dlat, dlng := r(a.lat-b.lat), r(a.lng-b.lng)
	c, d := r(a.lat), r(b.lat)

	x := sin(dlat/2)*sin(dlat/2) + cos(c)*cos(d)*sin(dlng/2)*sin(dlng/2)
	y := 2 * atan2(sqrt(x), sqrt(1-x))
	v := er * y
	return v
}

// kd.Point methods ....

// Dim ...
func (ll *item) Dim() int { return 2 }

// GetValue ...
func (ll *item) GetValue(dim int) float64 {
	if dim == 0 {
		return ll.lat
	}
	return ll.lng
}

// Distance ...
func (ll *item) Distance(p point) float64 {
	var cos, atan2, sqrt, sin = math.Cos, math.Atan2, math.Sqrt, math.Sin
	alat, alng := ll.lat, ll.lng
	blat, blng := p.GetValue(0), p.GetValue(1)
	dlat, dlng := r(alat-blat), r(alng-blng)
	c, d := r(alat), r(blat)

	x := sin(dlat/2)*sin(dlat/2) + cos(c)*cos(d)*sin(dlng/2)*sin(dlng/2)
	y := 2 * atan2(sqrt(x), sqrt(1-x))
	v := er * y
	return v
}

// PlaneDistance ...
// Return the distance between the point and the plane X_{dim}=val
func (ll *item) PlaneDistance(val float64, dim int) float64 {
	return math.Abs(r(ll.GetValue(dim)-val)) * er
}
