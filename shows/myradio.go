package shows

import (
	"strings"

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
		return nil, err
	}
	return &Show{
		ID:        uint(ts.TimeslotID),
		StartTime: ts.StartTime,
		EndTime:   ts.StartTime.Add(ts.Duration),
	}, nil
}

func (m *MyRadioShowProvider) GetShow(id uint) (*Show, error) {
	ts, err := m.s.GetTimeslot(int(id))
	if err != nil {
		// I don't like this
		if strings.Contains(err.Error(), "HTTP 400") {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &Show{
		ID:        uint(ts.TimeslotID),
		StartTime: ts.StartTime,
		EndTime:   ts.StartTime.Add(ts.Duration),
	}, nil
}
