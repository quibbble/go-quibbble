package go_boardgame_networking

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func byteChanIsClosed(ch <-chan []byte) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

func unmarshalJSONRequestBody(r *http.Request, output interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("invalid request body")
	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &output); err != nil {
		return err
	}
	return nil
}

func contains(items []string, item string) bool {
	for _, it := range items {
		if it == item {
			return true
		}
	}
	return false
}
