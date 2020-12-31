package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/UniversityRadioYork/TimeMachine/recorder"
	"github.com/UniversityRadioYork/TimeMachine/shows"
	"github.com/UniversityRadioYork/myradio-go"
	"github.com/gorilla/mux"
)

var rec *recorder.IcecastPullRecorder

func checkShowLoop(h *HandlerContext) {
	attempt := 0
	var currentShow shows.Show
	ctx := context.Background()
	var cancelRec context.CancelFunc
	for {
		show, err := h.ShowProvider.GetCurrentShow()
		if err != nil {
			attempt++
			if attempt == 5 {
				panic(err)
			} else {
				continue
			}
		}
		attempt = 0
		if currentShow.ID != show.ID {
			// Cancel the current recording and start a new one
			if cancelRec != nil {
				cancelRec()
			}
			log.Printf("starting to record show %d\n", show.ID)
			newRec, err := recorder.NewIcecastPullRecorder("https://audio.ury.org.uk/live-high", show.ID)
			if err != nil {
				panic(err)
			}
			rec = newRec
			// Create a new cancel context
			var recCtx context.Context
			recCtx, cancelRec = context.WithCancel(ctx)
			go func() {
				recErr := rec.Record(recCtx)
				if recErr != nil && !errors.Is(recErr, context.Canceled) {
					panic(err)
				}
			}()
			currentShow = *show
		}
		time.Sleep(1 * time.Second)
	}
}

const useMyRadio = false

func main() {
	port := flag.Int("port", 3958, "Port to listen on")

	flag.Parse()

	if _, err := os.Stat("show_data"); os.IsNotExist(err) {
		err = os.Mkdir("show_data", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	var provider shows.ShowProvider
	if useMyRadio {
		myr, err := myradio.NewSessionFromKeyFile()
		if err != nil {
			log.Fatal(err)
		}
		provider, err = shows.NewMyRadioShowProvider(myr)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		provider = &shows.DummyShowProvider{}
	}

	h := HandlerContext{
		ShowProvider: provider,
	}

	go checkShowLoop(&h)

	r := mux.NewRouter()

	r.HandleFunc("/tm/v1/show/{id}", h.HandleGetShow)
	r.HandleFunc("/tm/v1/show/{id}/stream", h.HandleGetShowStream)

	http.Handle("/", r)

	log.Printf("Listening on %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *port), nil))
}
