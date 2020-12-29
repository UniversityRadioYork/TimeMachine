package shows

import "time"

type Show struct {
	ID        uint
	StartTime time.Time
	EndTime   time.Time
}

type ShowProvider interface {
	GetCurrentShow() (*Show, error)
	GetShow(id uint) (*Show, error)
}
