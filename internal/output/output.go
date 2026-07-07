// Package output renders command results as a human-readable table (default),
// JSON or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// FormatList are the accepted --output values.
var FormatList = []string{"table", "json", "yaml"}

// Formats lists the accepted values for the --output flag, for help and
// error texts.
var Formats = strings.Join(FormatList, ", ")

// Valid reports whether format is an accepted --output value.
func Valid(format string) bool {
	return format == "" || slices.Contains(FormatList, format)
}

// Render writes v to w in the requested format. For "table" it calls table,
// which renders the human-readable representation.
func Render(w io.Writer, format string, v any, table func(w io.Writer) error) error {
	switch format {
	case "", "table":
		return table(w)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case "yaml":
		// The generated API models carry only json tags, which yaml.v3
		// ignores — encoding them directly would mangle every key
		// (authproviderid instead of auth_provider_id) and drop omitempty.
		// Round-trip through JSON so YAML uses the same field names as JSON.
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var generic any
		if err := json.Unmarshal(jsonBytes, &generic); err != nil {
			return err
		}
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		return enc.Encode(generic)
	default:
		return fmt.Errorf("unknown output format %q (expected one of: %s)", format, Formats)
	}
}

// Table renders rows under a header with aligned columns, for list output.
func Table(w io.Writer, header []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

// KVWriter renders aligned key/value rows, for single-object output.
type KVWriter struct {
	tw *tabwriter.Writer
}

func NewKVWriter(w io.Writer) *KVWriter {
	return &KVWriter{tw: tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)}
}

// Row adds one key/value row. Pointer values (the generated API models use
// them for optional fields) are dereferenced; nil renders as empty rather
// than as an address.
func (k *KVWriter) Row(key string, value any) {
	if v := reflect.ValueOf(value); v.Kind() == reflect.Pointer {
		if v.IsNil() {
			value = ""
		} else {
			value = v.Elem().Interface()
		}
	}
	fmt.Fprintf(k.tw, "%s:\t%v\n", key, value)
}

func (k *KVWriter) Flush() error {
	return k.tw.Flush()
}
