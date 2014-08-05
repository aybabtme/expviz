package main

import (
	"github.com/pkg/browser"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {

	l, err := net.Listen("tcp4", ":0")
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

	err = http.Serve(l, http.FileServer(http.Dir("static/")))
	if err != nil {
		log.Fatal(err)
	}
}
