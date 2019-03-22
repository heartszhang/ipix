package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	inf "fun.tv/nara/configure"
	hu "fun.tv/nara/httputil"
	kd "github.com/hongshibao/go-kdtree"
)

type (
	point = kd.Point
	tree  = *kd.KDTree
)

var (
	be   = binary.BigEndian
	le   = binary.LittleEndian
	read = binary.Read
	do   = hu.PathDo
	j    = hu.ContentJSON
)

const null = "\000"

func main() {
	file, err := os.Open(config.dat)
	panice(err)

	var (
		addr  uint32
		owner uint32
	)
	err = read(file, le, &addr)
	panice(err)
	err = read(file, le, &owner)
	panice(err)

	b := time.Now()
	_, err = file.Seek(int64(addr), 1)
	panice(err)
	dict := make([]byte, owner-addr)
	_, err = file.Read(dict)
	panice(err)

	_, err = file.Seek(8, 0) // 回到ip区间起始地址
	panice(err)

	var points []point
	for r := range records(file, addr) {
		r.Addr -= addr
		_, created := config.addrs[r.Addr] // 根据地址是否注册过判断，是否需要加入kdtree
		i := toitem(r, dict)
		config.records = append(config.records, i)
		if !created && i.Code == "CN" {
			points = append(points, i)
		}
	}
	dict = nil
	file.Close()
	log.Println(time.Since(b), "uniqs", len(points), config.dat)

	config.tree = kd.NewKDTree(points) // very slow
	log.Println(time.Since(b), "kdtree ctor")

	hu.Serve(config.addr, handles())
}
func records(file io.Reader, addr uint32) <-chan *record {
	pipe := make(chan *record, 128)
	go func() {
		reader := bufio.NewReader(file)
		for i := uint32(0); i < addr; i += 108 {
			var r record
			if err := read(reader, le, &r); err == nil {
				pipe <- &r
			}
		}
		close(pipe)
	}()

	return pipe
}
func handles() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v39/ip/s.json", do(j, ip))
	mux.HandleFunc("/v39/geo/n.json", do(j, nearest)) //location=lat,lng

	return mux
}
func knnx(lat, lng float64) (points []kd.Point) {
	points = config.tree.KNN(&item{lat: lat, lng: lng}, 1)
	return
}
func nearest(w http.ResponseWriter, r *http.Request) {
	loc := r.FormValue("location")
	ll := strings.Split(loc, ",")
	if len(ll) != 2 {
		return
	}
	var lat, lng float64

	lat, err := strconv.ParseFloat(ll[0], 64)
	if err == nil {
		lng, err = strconv.ParseFloat(ll[1], 64)
	}
	panice(err)

	var point *item
	if points := knnx(lat, lng); len(points) > 0 {
		point, _ = points[0].(*item)
	}
	err = json.NewEncoder(w).Encode(point)
	panice(err)
}
func ip(w http.ResponseWriter, r *http.Request) {
	sip := r.FormValue("ip")
	iip := be.Uint32(net.ParseIP(sip).To4())
	idx := sort.Search(len(config.records), func(i int) bool {
		return config.records[i].max >= iip
	})
	var ret *item
	if idx < len(config.records) && config.records[idx].min <= iip { // founded
		ret = config.records[idx]
	}
	json.NewEncoder(w).Encode(ret)
}

func init() {
	config.addrs = map[uint32]*address{}

	flag.StringVar(&config.addr, "addr", inf.Conf.IPListen, "")
	flag.StringVar(&config.dat, "dat", inf.Conf.IPDat, "")
	flag.Parse()
}

var config struct {
	addr    string
	dat     string
	tree    tree
	addrs   map[uint32]*address
	records []*item
}

func panice(err error) {
	if err != nil {
		panic(err)
	}
}
