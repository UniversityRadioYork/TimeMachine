package shows

import (
	"strings"
	"time"

	"github.com/UniversityRadioYork/myradio-go"
)

type MyRadioShowProvider struct {
	s *myradio.Session
}

func NewMyRadioShowProvider(session *myradio.Session) (*MyRadioShowProvider, error) {
	return &MyRadioShowProvider{
		s: session,
	}, nil
}

func (m *MyRadioShowProvider) GetCurrentShow() (*Show, error) {
	ts, err := m.s.GetCurrentTimeslot()
	if err != nil {
		// Failed to get a show, it's likely jukebox or off air.
		// Make a fake show that begins on the hour and ends at the next.
		now := time.Now().UTC()
		startOfHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
		return &Show{
			ID:        uint(0),
			StartTime: startOfHour,
			EndTime:   startOfHour.Add(time.Hour),
		}, nil
	}
	return &Show{
		ID:        uint(ts.TimeslotID),
		StartTime: ts.StartTime,
		EndTime:   ts.StartTime.Add(ts.Duration),
	}, nil
}

func (m *MyRadioShowProvider) GetShow(startTime uint) (*Show, error) {
	unix := startTime * SECONDS_IN_HOUR
	ts, err := m.s.GetCurrentTimeslotAtTime(int(unix) + 1) // Add 1 sec so the API doesn't return the last show

	showTime := time.Unix(int64(unix), 0)
	if err != nil {
		// I don't like this
		if strings.Contains(err.Error(), "cannot parse \"\"") {
			// There's no show here. Return a hourly chunk
			return &Show{
				ID:        uint(0),
				StartTime: showTime,
				EndTime:   showTime.Add(time.Hour),
			}, nil
		} else {
			// API didn't like you
			return nil, err
		}
	}
	return &Show{
		ID:        uint(ts.TimeslotID),
		StartTime: ts.StartTime,
		EndTime:   ts.StartTime.Add(ts.Duration),
	}, nil
}
