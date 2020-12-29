package shows

import "time"

const SECONDS_IN_HOUR = 60 * 60

type DummyShowProvider struct{}

func (d *DummyShowProvider) GetCurrentShow() (*Show, error) {
	now := time.Now().UTC()
	showTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return &Show{
		ID:        uint(showTime.Unix() / SECONDS_IN_HOUR),
		StartTime: showTime,
		EndTime:   showTime.Add(time.Hour),
	}, nil
}

func (d *DummyShowProvider) GetShow(id uint) (*Show, error) {
	unix := id * SECONDS_IN_HOUR
	showTime := time.Unix(int64(unix), 0)
	return &Show{
		ID:        id,
		StartTime: showTime,
		EndTime:   showTime.Add(time.Hour),
	}, nil
}
