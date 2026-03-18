package util

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	Default PrettyFormat = 0
	Compact PrettyFormat = 1
)

type PrettyFormat int

type Error struct {
	Error string
}

// Format a using json tags and specified options.
// val may be either a json string or byte slice or any type which can be marshalled to json data.
func Pretty(val any, options ...PrettyFormat) string {
	format := Default
	for _, opt := range options {
		format = opt
	}
	var b strings.Builder
	switch v := val.(type) {
	case []byte:
		pretty(&b, unmarshalJSON(v), "", format)
	case string:
		pretty(&b, unmarshalJSON([]byte(v)), "", format)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			data = []byte(`{"error":` + strconv.Quote(err.Error()) + `})`)
		}
		pretty(&b, unmarshalJSON(data), "", format)
	}
	return b.String()
}

// Log msg followed by Pretty(val) if logrus debug level is set.
func LogDebug(msg string, val any) {
	if log.GetLevel() >= log.DebugLevel {
		log.Debug(msg, Pretty(val))
	}
}

// Log msg followed by Pretty(val) if logrus trace level is set.
func LogTrace(msg string, val any) {
	if log.GetLevel() >= log.TraceLevel {
		log.Trace(msg, Pretty(val))
	}
}

func pretty(w io.Writer, val any, indent string, format PrettyFormat) {
	switch t := val.(type) {
	case nil:
		fmt.Fprint(w, "nil")
	case Error:
		fmt.Fprint(w, "error: "+t.Error)
	case bool:
		fmt.Fprint(w, strconv.FormatBool(t))
	case float64:
		fmt.Fprint(w, strconv.FormatFloat(t, 'g', -1, 64))
	case string:
		fmt.Fprint(w, strconv.Quote(t))
	case []any:
		fmt.Fprint(w, "[")
		for i, elem := range t {
			separator(w, indent+"  ", i == 0, format)
			pretty(w, elem, indent+"  ", format)
		}
		separator(w, indent, true, format)
		fmt.Fprint(w, "]")
	case map[string]any:
		fmt.Fprint(w, "{")
		keys := make([]string, 0, len(t))
		for key := range t {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			separator(w, indent+"  ", i == 0, format)
			fmt.Fprint(w, key, ": ")
			pretty(w, t[key], indent+"  ", format)
		}
		separator(w, indent, true, format)
		fmt.Fprint(w, "}")
	default:
		panic(fmt.Errorf("invalid type %T from json.Unmarshal", val))
	}
}

func separator(w io.Writer, indent string, skip bool, format PrettyFormat) {
	if format == Compact {
		if !skip {
			fmt.Fprint(w, ", ")
		}
	} else {
		if !skip {
			fmt.Fprint(w, ",", indent)
		}
		fmt.Fprint(w, "\n", indent)
	}
}

func unmarshalJSON(data []byte) (v any) {
	if err := json.Unmarshal(data, &v); err == nil {
		return v
	} else {
		return Error{Error: err.Error()}
	}
}
