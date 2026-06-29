package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

const contractVersion = "v1"

type Envelope struct {
	OK    bool         `json:"ok"`
	Data  any          `json:"data"`
	Error *ErrorDetail `json:"error"`
	Meta  Meta         `json:"meta"`
}

type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type Meta struct {
	Version string `json:"version"`
}

func successEnvelope(data any) Envelope {
	return Envelope{
		OK:    true,
		Data:  data,
		Error: nil,
		Meta:  Meta{Version: contractVersion},
	}
}

func errorEnvelope(code ErrorCode, message string) Envelope {
	return Envelope{
		OK:   false,
		Data: nil,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
		},
		Meta: Meta{Version: contractVersion},
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
