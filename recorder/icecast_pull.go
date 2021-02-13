package recorder

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

type IcecastPullRecorder struct {
	Source string
	ShowID string
	client *http.Client
}

func NewIcecastPullRecorder(source string, id string) (*IcecastPullRecorder, error) {
	return &IcecastPullRecorder{
		Source: source,
		ShowID: id,
		client: &http.Client{},
	}, nil
}

func (i *IcecastPullRecorder) Record(ctx context.Context) error {
	outFile, err := os.OpenFile(fmt.Sprintf("show_data/%s.mp3", i.ShowID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	req, err := http.NewRequest("GET", i.Source, nil)
	if err != nil {
		return err
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	req.Header.Set("icy-metadata", "0") // don't put guff in my stream
	res, err := i.client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("icecast gave us a %d", res.StatusCode)
	}
	defer res.Body.Close()
	_, err = io.Copy(outFile, res.Body)
	return err
}
