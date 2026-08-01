package elasticsearch

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

type Elasticsearch struct {
	client *elasticsearch.Client
}

type Config struct {
	Addresses          []string
	Username           string
	Password           string
	InsecureSkipVerify bool
}

const elasticsearchTimeout = 3 * time.Second

func New(cfg Config) (*Elasticsearch, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   elasticsearchTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify},
			TLSHandshakeTimeout:   elasticsearchTimeout,
			ResponseHeaderTimeout: elasticsearchTimeout,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Elasticsearch{client: client}, nil
}

func (e *Elasticsearch) Client() *elasticsearch.Client {
	return e.client
}
