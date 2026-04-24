package output

import (
	"encoding/json"
	"io"

	"sf-cli/internal/model"
)

func WriteJSON(w io.Writer, envelope model.Envelope) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(envelope)
}
