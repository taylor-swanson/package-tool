// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
)

func JSON(w io.Writer, results map[string]Result) error {
	type Report struct {
		Time    string            `json:"time"`
		Results map[string]Result `json:"results"`
	}

	r := Report{
		Time:    time.Now().Format(time.RFC3339),
		Results: results,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	return enc.Encode(r)
}

func Text(w io.Writer, results map[string]Result) error {
	return text(w, results, false)
}

func ColorText(w io.Writer, results map[string]Result) error {
	return text(w, results, true)
}

func text(w io.Writer, results map[string]Result, wantColor bool) error {
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	bold := color.New(color.Bold)
	if !wantColor {
		red.DisableColor()
		green.DisableColor()
		bold.DisableColor()
	}

	var keys []string
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var err error
	for _, k := range keys {
		if _, err = bold.Fprintf(w, "-- %s %s\n", k, strings.Repeat("-", 76-len(k))); err != nil {
			return err
		}

		result := results[k]

		if _, err = bold.Fprintln(w, "  Findings"); err != nil {
			return err
		}
		if len(result.Findings) > 0 {
			for _, d := range result.Findings {
				if _, err = fmt.Fprint(w, "    - "); err != nil {
					return err
				}
				if _, err = red.Fprintf(w, "%s", d.Message); err != nil {
					return err
				}
				if _, err = fmt.Fprintf(w, " (%s)\n", d.Category); err != nil {
					return err
				}
				if d.Pos.File != "" {
					if _, err = bold.Fprintf(w, "      %s\n", d.Pos); err != nil {
						return err
					}
				}
				if len(d.Related) > 0 {
					for _, r := range d.Related {
						if _, err = fmt.Fprint(w, "      "); err != nil {
							return err
						}
						if _, err = red.Fprintf(w, "%s", r.Message); err != nil {
							return err
						}
						if r.Pos.File != "" {
							if _, err = bold.Fprintf(w, " %s", r.Pos); err != nil {
								return err
							}
						}
						if _, err = fmt.Println(); err != nil {
							return err
						}
					}
				}
			}
		} else {
			if _, err = green.Fprintln(w, "    No findings"); err != nil {
				return err
			}
		}

		if len(result.Fixes) > 0 {
			if _, err = bold.Fprintln(w, "  Fixes"); err != nil {
				return err
			}
			for _, d := range result.Fixes {
				if _, err = fmt.Fprint(w, "    - "); err != nil {
					return err
				}
				if _, err = green.Fprintf(w, "%s", d.Message); err != nil {
					return err
				}
				if _, err = fmt.Fprintf(w, " (%s)\n", d.Category); err != nil {
					return err
				}
			}
		}

		if _, err = fmt.Fprintln(w); err != nil {
			return err
		}
	}

	return nil
}
