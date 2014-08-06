package main

import (
	"expvar"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	tick := time.NewTicker(time.Millisecond * 20)

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	f := expvar.NewFloat("im.a.random.float")
	i := expvar.NewInt("im.a.random.int")
	s := expvar.NewString("im.a.random.string")
	m := expvar.NewMap("im.a.random.map")

	var derp []byte
	for _ = range tick.C {
		derp = make([]byte, rand.Int63n(1000000))

		f.Set(rand.Float64())
		i.Set(rand.Int63n(42))
		s.Set(fmt.Sprintf("derp %d", rand.Int63n(1000000)))

		m.AddFloat("a.float", rand.Float64())
		m.Add("a.int", rand.Int63n(42))

		_ = derp
	}
}
