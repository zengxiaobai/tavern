package remote

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/omalloc/tavern/contrib/config"
)

var _ config.Source = (*remotefile)(nil)

type remotefile struct {
	url        string
	httpClient *http.Client
}

// NewSource new a file source.
func NewSource(url string) config.Source {
	return &remotefile{
		url: url,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				MaxIdleConns:        5,
				MaxIdleConnsPerHost: 5,
			},
		},
	}
}

// Load implements config.Source.
func (f *remotefile) Load() ([]*config.KeyValue, error) {
	req, err := http.NewRequest(http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tavern/agent")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote config: %s", resp.Status)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return []*config.KeyValue{
		{
			Key:   "remote",
			Value: buf,
		},
	}, nil
}

// Watch implements config.Source.
func (f *remotefile) Watch() (config.Watcher, error) {
	panic("unimplemented")
}
