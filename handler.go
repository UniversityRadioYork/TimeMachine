package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/UniversityRadioYork/TimeMachine/shows"
	"github.com/gorilla/mux"
	"github.com/tcolgate/mp3"
)

type HandlerContext struct {
	ShowProvider shows.ShowProvider
}

type PageData struct {
	PageTitle string
	APIKey    string
}

func (h *HandlerContext) HandleUIRoot(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("ui/root.tmpl"))
	data := PageData{
		PageTitle: "URYPlayer Rewind",
		APIKey:    "rewind",
	}
	tmpl.Execute(w, data)

}

func (h *HandlerContext) HandleGetShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	startTime, err := strconv.ParseUint(vars["startTime"], 10, 32) // In hours since epoch
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("TM001 invalid startTime"))
		return
	}

	show, err := h.ShowProvider.GetShow(uint(startTime))
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte("TM501 something exploded!"))
		return
	}

	showDuration := show.EndTime.Sub(show.StartTime)
	available := time.Now().Before(show.EndTime.Add((showDuration)))

	var playable float64
	if available {
		var filename string
		if show.ID != 0 {
			filename = "timeslotid-" + fmt.Sprint(show.ID)
		} else {
			filename = "hour-" + fmt.Sprint(show.StartTime.Unix()/SECONDS_IN_HOUR)
		}

		file, err := os.Open(fmt.Sprintf("show_data/%s.mp3", filename))
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
	startTime, err := strconv.ParseUint(vars["startTime"], 10, 32) // In hours since epoch
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("TM001 invalid startTime"))
		return
	}

	show, err := h.ShowProvider.GetShow(uint(startTime))
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

	var filename string
	if show.ID != 0 {
		filename = "timeslotid-" + fmt.Sprint(show.ID)
	} else {
		filename = "hour-" + fmt.Sprint(show.StartTime.Unix()/SECONDS_IN_HOUR)
	}

	showFile, err := os.OpenFile(fmt.Sprintf("show_data/%s.mp3", filename), os.O_RDONLY, 0)
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

	BUFFERSIZE := 1024
	buf := make([]byte, BUFFERSIZE)
	attempt := 0
	for {
		n, err := showFile.Read(buf)
		if err != nil && err != io.EOF {
			return //err
		}
		if n == 0 {
			// There's no new bytes now, but there be some new
			if attempt > 4 {
				// Attempts to get new bytes from the file failed.
				// Likely means the recording has finished.
				fmt.Println("Reached end of recording.")
				return
			}

			attempt = attempt + 1
			time.Sleep(1 * time.Second)
			continue
		}

		if _, err := w.Write(buf[:n]); err != nil {
			fmt.Println("Client probably went away")
			return
		}
		attempt = 0
	}
}
