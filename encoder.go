package zapang

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// --- Console encoder: human-readable fields + verbose error block ---

// consoleEncoder wraps a zapcore.Encoder to:
//   - intercept "errorVerbose" and render it as a colored multi-line block
//   - reformat JSON fields blob as key=value pairs
type consoleEncoder struct {
	zapcore.Encoder
	verbose string
}

func newConsoleEncoder(inner zapcore.Encoder) *consoleEncoder {
	return &consoleEncoder{Encoder: inner}
}

func (e *consoleEncoder) Clone() zapcore.Encoder {
	return &consoleEncoder{Encoder: e.Encoder.Clone()}
}

func (e *consoleEncoder) AddString(key, val string) {
	if key == "errorVerbose" {
		e.verbose = val
		return
	}
	e.Encoder.AddString(key, val)
}

func (e *consoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	var verbose string

	// Replace ErrorType fields with plain strings to prevent inline errorVerbose.
	modified := make([]zapcore.Field, 0, len(fields))
	for _, f := range fields {
		if f.Type == zapcore.ErrorType {
			if err, ok := f.Interface.(error); ok {
				modified = append(modified, zap.String(f.Key, err.Error()))
				v := fmt.Sprintf("%+v", err)
				if v != err.Error() {
					verbose = v
				}
				continue
			}
		}
		modified = append(modified, f)
	}

	if e.verbose != "" {
		verbose = e.verbose
		e.verbose = ""
	}

	buf, err := e.Encoder.EncodeEntry(entry, modified)
	if err != nil {
		return buf, err
	}

	data := buf.String()
	buf.Reset()

	// Reformat JSON fields blob as key=value pairs.
	data = reformatJSONFields(data)

	if verbose == "" {
		buf.AppendString(data)
		return buf, nil
	}

	buf.AppendString(strings.TrimRight(data, "\n"))
	buf.AppendString("\n")
	buf.AppendString(colorizeVerbose(verbose))
	buf.AppendString("\n")
	return buf, nil
}

// reformatJSONFields finds the trailing JSON object in the first line
// and replaces it with tab-separated key=value pairs.
func reformatJSONFields(data string) string {
	// Split first line from the rest (stacktrace etc.)
	firstLine, rest, hasRest := strings.Cut(data, "\n")

	idx := strings.LastIndex(firstLine, "\t{\"")
	if idx < 0 {
		return data
	}

	jsonStr := firstLine[idx+1:]
	prefix := firstLine[:idx]

	var fields map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
		return data
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(prefix)
	for _, k := range keys {
		b.WriteByte('\t')
		b.WriteString(k)
		b.WriteByte('=')
		fmt.Fprint(&b, fields[k])
	}
	b.WriteByte('\n')

	if hasRest {
		b.WriteString(rest)
	}

	return b.String()
}

// --- Export encoder: strips errorVerbose from JSON output ---

// exportEncoder wraps a JSON encoder to drop the errorVerbose field.
type exportEncoder struct {
	zapcore.Encoder
}

func newExportEncoder(inner zapcore.Encoder) *exportEncoder {
	return &exportEncoder{Encoder: inner}
}

func (e *exportEncoder) Clone() zapcore.Encoder {
	return &exportEncoder{Encoder: e.Encoder.Clone()}
}

func (e *exportEncoder) AddString(key, val string) {
	if key == "errorVerbose" {
		return
	}
	e.Encoder.AddString(key, val)
}

func (e *exportEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// Replace ErrorType with plain String to prevent errorVerbose generation.
	modified := make([]zapcore.Field, 0, len(fields))
	for _, f := range fields {
		if f.Type == zapcore.ErrorType {
			if err, ok := f.Interface.(error); ok {
				modified = append(modified, zap.String(f.Key, err.Error()))
				continue
			}
		}
		modified = append(modified, f)
	}
	return e.Encoder.EncodeEntry(entry, modified)
}

// --- Formatting helpers ---

const (
	ansiReset   = "\033[0m"
	ansiBoldRed = "\033[1;31m"
	ansiDim     = "\033[2m"
)

func colorizeVerbose(verbose string) string {
	lines := strings.Split(verbose, "\n")
	var b strings.Builder
	b.Grow(len(verbose) + len(lines)*16)

	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		switch {
		case i == 0 || strings.HasPrefix(line, "  - "):
			b.WriteString(ansiBoldRed)
			b.WriteString(line)
			b.WriteString(ansiReset)
		default:
			b.WriteString(ansiDim)
			b.WriteString(line)
			b.WriteString(ansiReset)
		}
	}

	return b.String()
}
