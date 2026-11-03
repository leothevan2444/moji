package cardigann

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/ohler55/ojg/jp"
)

// Keep object member order for JObject child enumeration and nested ToString.
// jp.Keyed lets the existing JSONPath evaluator consume this local representation.
type joinObject struct {
	keys   []string
	values map[string]any
}

func (o *joinObject) ValueForKey(key string) (any, bool) {
	value, ok := o.values[key]
	return value, ok
}
func (o *joinObject) SetValueForKey(key string, value any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}
func (o *joinObject) RemoveValueForKey(key string) {
	delete(o.values, key)
	for i, name := range o.keys {
		if name == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}
func (o *joinObject) Keys() []string { return append([]string(nil), o.keys...) }

func readJoinValue(ctx context.Context, decoder *json.Decoder, depth int, remaining *int) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	*remaining--
	if depth > 64 || *remaining < 0 {
		return nil, fmt.Errorf("jsonjoinarray node/depth budget exceeded")
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		object := &joinObject{values: map[string]any{}}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("invalid JSON object key")
			}
			value, err := readJoinValue(ctx, decoder, depth+1, remaining)
			if err != nil {
				return nil, err
			}
			object.SetValueForKey(name, value)
		}
		_, err = decoder.Token()
		return object, err
	case json.Delim('['):
		array := []any{}
		for decoder.More() {
			value, err := readJoinValue(ctx, decoder, depth+1, remaining)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		_, err = decoder.Token()
		return array, err
	default:
		return token, nil
	}
}

func jsonJoinArray(ctx context.Context, data, selector, separator string) (string, error) {
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()
	remaining := 50000
	root, err := readJoinValue(ctx, decoder, 0, &remaining)
	if err != nil {
		return "", fmt.Errorf("jsonjoinarray requires a bounded JSON object: %w", err)
	}
	if _, ok := root.(*joinObject); !ok {
		return "", fmt.Errorf("jsonjoinarray requires a JSON object")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", fmt.Errorf("trailing jsonjoinarray input")
	}
	path, err := jp.ParseString(selector)
	if err != nil {
		return "", err
	}
	selected := path.Get(root)
	if len(selected) != 1 {
		return "", fmt.Errorf("jsonjoinarray path must select one token")
	}
	var parts []string
	used := 0
	appendPart := func(text string) error {
		extra := len(text)
		if len(parts) > 0 {
			extra += len(separator)
		}
		if extra > MaxTemplateBytes-used {
			return fmt.Errorf("jsonjoinarray output exceeds byte budget")
		}
		used += extra
		parts = append(parts, text)
		return nil
	}
	switch value := selected[0].(type) {
	case []any:
		for _, item := range value {
			text, err := joinTokenText(ctx, item)
			if err != nil {
				return "", err
			}
			if err := appendPart(text); err != nil {
				return "", err
			}
		}
	case *joinObject:
		for _, key := range value.keys {
			text, err := joinJSONText(ctx, value.values[key])
			if err != nil {
				return "", err
			}
			if err := appendPart(joinQuote(key) + ": " + text); err != nil {
				return "", err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	result := strings.Join(parts, separator)
	if len(result) > MaxTemplateBytes {
		return "", fmt.Errorf("jsonjoinarray output exceeds byte budget")
	}
	return result, nil
}

func joinTokenText(ctx context.Context, value any) (string, error) {
	switch typed := value.(type) {
	case *joinObject, []any:
		return joinJSONText(ctx, value)
	case json.Number:
		return joinNumberText(typed, false)
	default:
		return jsonScalarText(value), nil
	}
}

func joinQuote(value string) string {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.ReplaceAll(strings.TrimSuffix(out.String(), "\n"), "\u0085", `\u0085`)
}

func joinNumberText(value json.Number, literal bool) (string, error) {
	text := string(value)
	if !strings.ContainsAny(text, ".eE") {
		return text, nil
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
		return "", fmt.Errorf("invalid jsonjoinarray number")
	}
	format := byte('f')
	if magnitude := math.Abs(number); magnitude != 0 && (magnitude < 1e-4 || magnitude >= 1e17) {
		format = 'E'
	}
	text = strconv.FormatFloat(number, format, -1, 64)
	if literal && !strings.ContainsAny(text, ".E") {
		text += ".0"
	}
	return text, nil
}

func joinJSONText(ctx context.Context, value any) (string, error) {
	var out strings.Builder
	var write func(any, int) error
	write = func(value any, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		switch typed := value.(type) {
		case *joinObject:
			out.WriteByte('{')
			for i, key := range typed.keys {
				if i > 0 {
					out.WriteByte(',')
				}
				out.WriteString("\n" + strings.Repeat("  ", depth+1) + joinQuote(key) + ": ")
				if err := write(typed.values[key], depth+1); err != nil {
					return err
				}
			}
			if len(typed.keys) > 0 {
				out.WriteString("\n" + strings.Repeat("  ", depth))
			}
			out.WriteByte('}')
		case []any:
			out.WriteByte('[')
			for i, child := range typed {
				if i > 0 {
					out.WriteByte(',')
				}
				out.WriteString("\n" + strings.Repeat("  ", depth+1))
				if err := write(child, depth+1); err != nil {
					return err
				}
			}
			if len(typed) > 0 {
				out.WriteString("\n" + strings.Repeat("  ", depth))
			}
			out.WriteByte(']')
		case string:
			if date, ok := jsonDateJSON(typed); ok {
				typed = date
			}
			out.WriteString(joinQuote(typed))
		case json.Number:
			text, err := joinNumberText(typed, true)
			if err != nil {
				return err
			}
			out.WriteString(text)
		case bool:
			out.WriteString(strconv.FormatBool(typed))
		case nil:
			out.WriteString("null")
		}
		if out.Len() > MaxTemplateBytes {
			return fmt.Errorf("jsonjoinarray output exceeds byte budget")
		}
		return nil
	}
	if err := write(value, 0); err != nil {
		return "", err
	}
	return out.String(), nil
}
