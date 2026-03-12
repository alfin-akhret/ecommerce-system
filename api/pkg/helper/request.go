package helper

import (
	"encoding/json"
	"errors"
	"net/http"
)

func DecodeJSON(r *http.Request, dst any) error {

	if r.Body == nil {
		return errors.New("empty body")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}
