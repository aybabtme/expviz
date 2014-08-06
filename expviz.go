package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aybabtme/broadcaster"
	"github.com/pkg/browser"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

func main() {

	if len(os.Args) != 2 {
		log.Fatal("need an expvar HTTP endpoint")
	}

	var (
		netIface     string
		targetExpvar = os.Args[1]
	)

	flag.StringVar(&netIface, "interface", "127.0.0.1", "interface to listen on, by default on the private loopback interface")
	flag.Parse()

	bcast, err := pollTarget(targetExpvar)
	if err != nil {
		log.Fatalf("couldn't get expvar: %v", err)
	}

	serveToBrowser(netIface, websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		lstn := bcast.Listen()
		defer lstn.Close()

		for {
			e, more := lstn.Next()
			if !more {
				return
			}
			snap := e.(*snapshot)
			err := websocket.JSON.Send(conn, snap)
			if err != nil {
				log.Printf("failed to send snapshot to %q: %v", conn.RemoteAddr(), err)
				return
			}
		}
	}))

}

func serveToBrowser(netIface string, h http.Handler) {
	l, err := net.Listen("tcp4", netIface+":0")
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + l.Addr().String()

	time.AfterFunc(time.Millisecond*500, func() {
		err := browser.OpenURL(url)
		if err != nil {
			log.Printf("failed to open browser: %v", err)
			log.Printf("please visit %s", url)
		} else {
			log.Printf("browser window opened")
		}
	})

	mux := http.NewServeMux()
	mux.Handle("/ws", h)
	mux.Handle("/", http.FileServer(http.Dir("static/")))
	err = http.Serve(l, mux)
	if err != nil {
		log.Fatal(err)
	}
}

func pollTarget(target string) (broadcaster.Broadcaster, error) {
	endpoint := target + "/debug/vars"

	snap, err := fetchSnapshot(endpoint)
	if err != nil {
		return nil, err
	}

	bcast := broadcaster.NewBacklog(60 * 10)
	bcast.Send(snap)

	go func(bcast broadcaster.Broadcaster) {
		ticker := time.NewTicker(time.Second * 1)
		defer bcast.Close()
		defer ticker.Stop()
		for _ = range ticker.C {
			snap, err := fetchSnapshot(endpoint)
			if err != nil {
				log.Fatal(err)
			}
			bcast.Send(snap)
		}
	}(bcast)

	return bcast, nil
}

func fetchSnapshot(target string) (*snapshot, error) {
	resp, err := http.Get(target)
	if err != nil {
		return nil, fmt.Errorf("couldn't GET %q: %v", target, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status %q", resp.Status)
	}
	snap, err := fromReader(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error reading snapshot: %v", err)
	}
	return snap, nil
}

type snapshot struct {
	Time     time.Time        `json:"time"`
	Cmdline  []string         `json:"cmdline"`
	Memstats runtime.MemStats `json:"memstats"`

	Numbers    map[string]float64            `json:"numbers"`
	NumberMaps map[string]map[string]float64 `json:"number_maps"`
	Strings    map[string]string             `json:"strings"`
	StringMaps map[string]map[string]string  `json:"string_maps"`
}

func fromReader(r io.Reader) (*snapshot, error) {

	snap := &snapshot{
		Time:       time.Now(),
		NumberMaps: make(map[string]map[string]float64),
		StringMaps: make(map[string]map[string]string),
	}

	raw := make(map[string]interface{})
	err := json.NewDecoder(r).Decode(&raw)
	if err != nil {
		return nil, fmt.Errorf("decoding extra information: %v", err)
	}

	for k, v := range raw {
		switch k {
		case "cmdline":
			for _, val := range v.([]interface{}) {
				snap.Cmdline = append(snap.Cmdline, val.(string))
			}
			continue
		case "memstats":
			snap.Memstats = loadMemStats(v.(map[string]interface{}))
			continue
		}

		// switch val := v.(type) {
		// case float64:
		// 	snap.Numbers[k] = val
		// case string:
		// 	snap.Strings[k] = val
		// case map[string]interface{}:
		//           for subk, subv := range val {
		// 		switch subval := subv.(type) {
		// 		case float64:
		// 			snap.NumberMaps[k][subk] = subval
		// 		case string:
		// 			snap.StringMaps[k][subk] = subval
		// 		}
		// 	}
		// default:
		// 	log.Fatalf("not a supported type (%T): %q:%#v", v, k, v)
		// }
	}
	return snap, nil
}

func loadMemStats(values map[string]interface{}) runtime.MemStats {

	stats := runtime.MemStats{
		Alloc:      uint64(values["Alloc"].(float64)),
		TotalAlloc: uint64(values["TotalAlloc"].(float64)),
		Sys:        uint64(values["Sys"].(float64)),
		Lookups:    uint64(values["Lookups"].(float64)),
		Mallocs:    uint64(values["Mallocs"].(float64)),
		Frees:      uint64(values["Frees"].(float64)),

		HeapAlloc:    uint64(values["HeapAlloc"].(float64)),
		HeapSys:      uint64(values["HeapSys"].(float64)),
		HeapIdle:     uint64(values["HeapIdle"].(float64)),
		HeapInuse:    uint64(values["HeapInuse"].(float64)),
		HeapReleased: uint64(values["HeapReleased"].(float64)),
		HeapObjects:  uint64(values["HeapObjects"].(float64)),

		StackInuse:  uint64(values["StackInuse"].(float64)),
		StackSys:    uint64(values["StackSys"].(float64)),
		MSpanInuse:  uint64(values["MSpanInuse"].(float64)),
		MSpanSys:    uint64(values["MSpanSys"].(float64)),
		MCacheInuse: uint64(values["MCacheInuse"].(float64)),
		MCacheSys:   uint64(values["MCacheSys"].(float64)),
		BuckHashSys: uint64(values["BuckHashSys"].(float64)),
		GCSys:       uint64(values["GCSys"].(float64)),
		OtherSys:    uint64(values["OtherSys"].(float64)),

		NextGC:       uint64(values["NextGC"].(float64)),
		LastGC:       uint64(values["LastGC"].(float64)),
		PauseTotalNs: uint64(values["PauseTotalNs"].(float64)),
		PauseNs:      loadValues(values["PauseNs"].([]interface{})),
		NumGC:        uint32(values["NumGC"].(float64)),
		EnableGC:     values["EnableGC"].(bool),
		DebugGC:      values["DebugGC"].(bool),
	}

	for i, vals := range values["BySize"].([]interface{}) {
		v := vals.(map[string]interface{})
		stats.BySize[i] = struct {
			Size    uint32
			Mallocs uint64
			Frees   uint64
		}{
			Size:    uint32(v["Size"].(float64)),
			Mallocs: uint64(v["Mallocs"].(float64)),
			Frees:   uint64(v["Frees"].(float64)),
		}
	}

	return stats
}

func loadValues(vals []interface{}) [256]uint64 {
	var arr [256]uint64
	for i, v := range vals {
		arr[i] = uint64(v.(float64))
	}
	return arr
}
