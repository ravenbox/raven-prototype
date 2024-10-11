package main

import (
	"log"
	"net/http"

	"github.com/ravenbox/raven-prototype"
	"github.com/ravenbox/raven-prototype/pkg/sfu"
)

func main() {
	sfu := sfu.NewSFU()
	raven := raven.NewRaven(sfu)

	log.Println("Starting server...")

	err := http.ListenAndServe("127.0.0.1:8000", raven)
	if err != nil {
		log.Panicln("Bruh", err)
	}
}
