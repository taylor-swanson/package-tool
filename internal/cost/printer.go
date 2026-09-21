package cost

import (
	"encoding/json"
	"errors"
	"io"
)

func PrintJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	return enc.Encode(report)
}

func PrintText(w io.Writer, report *Report, wantColor bool) error {
	return errors.New("text output not yet implemented")
}
