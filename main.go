package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/UniversityRadioYork/TimeMachine/recorder"
	"github.com/UniversityRadioYork/TimeMachine/shows"
	"github.com/UniversityRadioYork/myradio-go"
	"github.com/gorilla/mux"
)

const SECONDS_IN_HOUR = 60 * 60

var rec *recorder.IcecastPullRecorder

func checkShowLoop(h *HandlerContext) {
	attempt := 0
	lastHour := -1
	var currentShow shows.Show
	ctx := context.Background()
	var cancelRec context.CancelFunc
	for {
		hours, _, _ := time.Now().Clock()
		if hours != lastHour {
			lastHour = hours
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
			if currentShow.StartTime != show.StartTime {
				// Cancel the current recording and start a new one
				if cancelRec != nil {
					cancelRec()
				}
				var filename string

				if show.ID != 0 {
					filename = "timeslotid-" + fmt.Sprint(show.ID)
				} else {
					filename = "hour-" + fmt.Sprint(show.StartTime.Unix()/SECONDS_IN_HOUR)
				}
				log.Printf("starting to record show: %s\n", filename)

				newRec, err := recorder.NewIcecastPullRecorder("https://audio.ury.org.uk/live-high", filename)
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

				// Right, now we've started a new recording, let's tidy up any old ones.

				// First, let's get a list of all of the files in the show_data directory.
				var files []string
				root := "show_data/"
				err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
					files = append(files, path)
					return nil
				})
				if err != nil {
					panic(err)
				}

				// For each of the two recording types we're interested in, make a list of their numbers.
				var slotTypes = []string{"timeslotid", "hour"}
				for _, slotType := range slotTypes {

					var fileNumbers []int
					for _, file := range files {
						res1 := strings.Split(file, "-")
						if len(res1) == 2 && root+slotType == res1[0] {
							numberStr := strings.Split(res1[1], ".")[0]
							var number int
							number, err = strconv.Atoi(numberStr)
							if err == nil {
								fileNumbers = append(fileNumbers, number)
							}

						}
					}

					// Now sort these numbers in incrementing order (oldest files first)
					sort.Ints(fileNumbers)

					// Now take all but the newest two (for current and previous show)
					// And delete the rest.
					if len(fileNumbers) > 2 {
						fileNumbersToRemove := fileNumbers[:len(fileNumbers)-2]

						for _, fileNumber := range fileNumbersToRemove {
							filename := fmt.Sprintf("show_data/%s-%d.mp3", slotType, fileNumber)
							fmt.Println("Removing old file: " + filename)
							e := os.Remove(filename)
							if e != nil {
								log.Fatal(e)
							}
						}
					}
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}

const useMyRadio = true

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

	r.HandleFunc("/", h.HandleUIRoot)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))
	r.HandleFunc("/tm/v1/show/{startTime}", h.HandleGetShow)
	r.HandleFunc("/tm/v1/show/{startTime}/stream", h.HandleGetShowStream)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf("0.0.0.0:%d", *port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Listening on %d\n", *port)
	log.Fatal(srv.ListenAndServe())
}
