package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/UniversityRadioYork/time-machine/shows"
	"github.com/gorilla/mux"
	"github.com/tcolgate/mp3"
)

type HandlerContext struct {
	ShowProvider shows.ShowProvider
}

func (h *HandlerContext) HandleGetShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("TM001 invalid id"))
		return
	}

	show, err := h.ShowProvider.GetShow(uint(id))
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte("TM510 something exploded!"))
		return
	}

	showDuration := show.EndTime.Sub(show.StartTime)
	available := time.Now().Before(show.EndTime.Add((showDuration)))

	var playable float64
	if available {
		file, err := os.Open(fmt.Sprintf("show_data/%d.mp3", show.ID))
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			w.Write([]byte("TM511 something exploded!"))
			return
		}
		decoder := mp3.NewDecoder(file)
		var frame mp3.Frame
		var skipped int
		for {
			err = decoder.Decode(&frame, &skipped)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				} else {
					log.Println(err)
					w.WriteHeader(500)
					w.Write([]byte("TM512 something exploded!"))
					return
				}
			}
			playable += frame.Duration().Seconds()
		}
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(struct {
		Show      shows.Show
		Available bool
		Playable  int
	}{
		Show:      *show,
		Available: available,
		Playable:  int(math.Floor(playable)),
	})
}

func (h *HandlerContext) HandleGetShowStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("TM001 invalid id"))
		return
	}

	show, err := h.ShowProvider.GetShow(uint(id))
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte("TM501 something exploded!"))
		return
	}

	showDuration := show.EndTime.Sub(show.StartTime)
	if time.Now().After(show.EndTime.Add(showDuration)) {
		w.WriteHeader(403)
		w.Write([]byte("TM002 rewind expired"))
	}

	var offset float64
	offsetQ := r.URL.Query().Get("offset")
	if offsetQ != "" {
		offset, err = strconv.ParseFloat(offsetQ, 10)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("TM003 invalid offset"))
			return
		}
	}

	showFile, err := os.OpenFile(fmt.Sprintf("show_data/%d.mp3", show.ID), os.O_RDONLY, 0)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte("TM502 something exploded!"))
		return
	}
	defer showFile.Close()

	// If offset is zero, we can just serve the file directly
	if offset != 0 {
		// Oh Lord Jesus
		decoder := mp3.NewDecoder(showFile)
		var currentPosition float64
		var lastFrameDuration float64
		var frame mp3.Frame
		var skipped int
		for {
			if currentPosition+lastFrameDuration > offset {
				// If we were to seek any further we'd pass the offset, start serving now
				break
			}
			err = decoder.Decode(&frame, &skipped)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// reached the end of the file => invalid offset
					w.WriteHeader(400)
					w.Write([]byte("TM004 invalid offset"))
				} else {
					log.Println(err)
					w.WriteHeader(500)
					w.Write([]byte("TM503 something exploded!"))
				}
				return
			}
			currentPosition += frame.Duration().Seconds()
		}
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Del("Content-Length")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	for {
		_, err = io.Copy(w, showFile)
		if err != nil {
			log.Printf("handler: io.Copy error %v\n", err)
			return
		}
	}
}
