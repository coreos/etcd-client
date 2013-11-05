package etcd

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
)

func (c *Client) Get(key string) ([]*Response, error) {
	c.cluster.RLock()
	logger.Debugf("get %s [%s]", key, c.cluster.Leader)
	c.cluster.RUnlock()
	resp, err := c.sendRequest("GET", path.Join("keys", key), "")

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

	return convertGetResponse(b)

}

// GetTo gets the value of the key from a given machine address.
// If the given machine is not available it returns an error.
// Mainly use for testing purpose
func (c *Client) GetFrom(key string, addr string) ([]*Response, error) {
	httpPath := c.createHttpPath(addr, path.Join(version, "keys", key))

	resp, err := c.httpClient.Get(httpPath)

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

	return convertGetResponse(b)
}

// Convert byte stream to response.
func convertGetResponse(b []byte) ([]*Response, error) {

	var results []*Response
	var result *Response

	err := json.Unmarshal(b, &result)

	if err != nil {
		err = json.Unmarshal(b, &results)

		if err != nil {
			return nil, err
		}

	} else {
		results = make([]*Response, 1)
		results[0] = result
	}
	return results, nil
}
