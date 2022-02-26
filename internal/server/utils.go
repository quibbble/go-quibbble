package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/unrolled/render"
)

func writeJSONResponse(render *render.Render, w http.ResponseWriter, statusCode int, responseModel interface{}) {
	if err := render.JSON(w, statusCode, responseModel); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
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
