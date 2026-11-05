package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/provsenya/avito/api"
)

const maxRequestBodySize = 1 << 20

func decodeCreateTripRequest(
	w http.ResponseWriter,
	r *http.Request,
) (api.CreateTripJSONRequestBody, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return api.CreateTripJSONRequestBody{}, errors.New("content type must be application/json")
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
	if err != nil {
		return api.CreateTripJSONRequestBody{}, fmt.Errorf("read request body: %w", err)
	}

	var request api.CreateTripJSONRequestBody
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return api.CreateTripJSONRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}

	if err := ensureJSONEnd(decoder); err != nil {
		return api.CreateTripJSONRequestBody{}, err
	}
	if err := validateRequiredJSONFields(body); err != nil {
		return api.CreateTripJSONRequestBody{}, err
	}

	return request, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func validateRequiredJSONFields(body []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("decode request fields: %w", err)
	}

	if err := requireFields(fields, "user_id", "driver_id", "start_point", "end_point", "price"); err != nil {
		return err
	}
	if err := requireCoordinateFields(fields["start_point"]); err != nil {
		return fmt.Errorf("start_point: %w", err)
	}
	if err := requireCoordinateFields(fields["end_point"]); err != nil {
		return fmt.Errorf("end_point: %w", err)
	}
	return nil
}

func requireCoordinateFields(raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return errors.New("must be an object")
	}
	return requireFields(fields, "latitude", "longitude")
}

func requireFields(fields map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field %q is required", name)
		}
	}
	return nil
}
