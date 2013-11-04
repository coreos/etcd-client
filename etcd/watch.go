package etcd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

type respAndErr struct {
	resp *http.Response
	err  error
}

// Errors introduced by the Watch command.
var (
	ErrWatchStoppedByUser = errors.New("Watch stopped by the user via stop channel")
)

// Watch any change under the given prefix.
// When a sinceIndex is given, watch will try to scan from that index to the last index
// and will return any changes under the given prefix during the history
// If a receiver channel is given, it will be a long-term watch. Watch will block at the
// channel. And after someone receive the channel, it will go on to watch that prefix.
// If a stop channel is given, client can close long-term watch using the stop channel

func (c *Client) Watch(prefix string, sinceIndex uint64, receiver chan *Response, stop chan bool) (*Response, error) {
	logger.Debugf("watch %s [%s]", prefix, c.cluster.Leader)
	if receiver == nil {
		return c.watchOnce(prefix, sinceIndex, stop)

	} else {
		for {
			resp, err := c.watchOnce(prefix, sinceIndex, stop)
			if resp != nil {
				sinceIndex = resp.Index + 1
				receiver <- resp
			} else {
				return nil, err
			}
		}
	}

	return nil, nil
}

// helper func
// return when there is change under the given prefix
func (c *Client) watchOnce(key string, sinceIndex uint64, stop chan bool) (*Response, error) {

	var resp *http.Response
	var err error

	if stop != nil {
		ch := make(chan respAndErr)
		fin := make(chan bool)

		go func() {
			resp, err = c.sendWatchRequest(key, sinceIndex)

			select {
			case ch <- respAndErr{resp, err}:
			case <-fin:
				resp.Body.Close()
			}
		}()

		// select at stop or continue to receive
		select {

		case res := <-ch:
			resp, err = res.resp, res.err

		case <-stop:
			close(fin)
			return nil, ErrWatchStoppedByUser
		}
	} else {
		resp, err = c.sendWatchRequest(key, sinceIndex)
	}

	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {

		return nil, handleError(b)
	}

	var result Response

	err = json.Unmarshal(b, &result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) sendWatchRequest(key string, sinceIndex uint64) (*http.Response, error) {
	if sinceIndex == 0 {
		resp, err := c.sendRequest("GET", path.Join("watch", key), "")
		return resp, err
	} else {
		v := url.Values{}
		v.Set("index", fmt.Sprintf("%v", sinceIndex))
		resp, err := c.sendRequest("POST", path.Join("watch", key), v.Encode())
		return resp, err
	}

}
