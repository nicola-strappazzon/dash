package elasticsearch

import (
	"context"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ThreadHot fetches _nodes/hot_threads as a raw text stream for the given
// node (empty string means every node). The caller must close the returned
// reader.
func (e *Elasticsearch) ThreadHot(ctx context.Context, node string) (io.ReadCloser, error) {
	requestOpts := []func(*esapi.NodesHotThreadsRequest){
		e.client.Nodes.HotThreads.WithContext(ctx),
	}
	if node != "" {
		requestOpts = append(requestOpts, e.client.Nodes.HotThreads.WithNodeID(node))
	}

	res, err := e.client.Nodes.HotThreads(requestOpts...)
	if err != nil {
		return nil, err
	}

	if res.IsError() {
		defer res.Body.Close()
		return nil, fmt.Errorf("unexpected response from _nodes/hot_threads: %s", res.String())
	}

	return res.Body, nil
}
