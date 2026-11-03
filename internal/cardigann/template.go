package cardigann

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dlclark/regexp2"
)

const MaxTemplateBytes = 1 << 20
const MaxTemplateSteps = 10000

type TemplateValue struct {
	Text  string
	Items []string
}
type Variables map[string]TemplateValue

func (v Variables) Clone() Variables {
	out := make(Variables, len(v))
	for k, value := range v {
		out[k] = value
	}
	return out
}

type Template struct{ nodes []templateNode }
type templateNode struct {
	kind         string
	text         string
	expr         expression
	children     []templateNode
	otherwise    []templateNode
	rangeIndex   string
	rangeElement string
}
type expression struct {
	kind string
	text string
	args []expression
}
type templateToken struct {
	text    string
	literal bool
}

func CompileTemplate(source string) (*Template, error) {
	if len(source) > MaxTemplateBytes {
		return nil, fmt.Errorf("template exceeds byte budget")
	}
	position := 0
	nodes, stop, err := parseTemplate(source, &position, 0)
	if err != nil {
		return nil, err
	}
	if stop != "" {
		return nil, fmt.Errorf("unexpected template %s", stop)
	}
	return &Template{nodes: nodes}, nil
}

func parseTemplate(source string, position *int, depth int) ([]templateNode, string, error) {
	if depth > 32 {
		return nil, "", fmt.Errorf("template nesting budget exceeded")
	}
	var nodes []templateNode
	for *position < len(source) {
		start := strings.Index(source[*position:], "{{")
		if start < 0 {
			nodes = append(nodes, templateNode{kind: "text", text: source[*position:]})
			*position = len(source)
			break
		}
		start += *position
		if start > *position {
			nodes = append(nodes, templateNode{kind: "text", text: source[*position:start]})
		}
		end := strings.Index(source[start+2:], "}}")
		if end < 0 {
			return nil, "", fmt.Errorf("unclosed template action")
		}
		end += start + 2
		action := strings.TrimSpace(source[start+2 : end])
		*position = end + 2
		if action == "else" || action == "end" {
			return nodes, action, nil
		}
		if action == "" {
			return nil, "", fmt.Errorf("empty template action")
		}
		node := templateNode{kind: "value"}
		for _, prefix := range []string{"if ", "range "} {
			if strings.HasPrefix(action, prefix) {
				node.kind = strings.TrimSpace(prefix)
				action = strings.TrimSpace(strings.TrimPrefix(action, prefix))
				break
			}
		}
		if node.kind == "range" {
			if assignment := strings.Index(action, ":="); assignment >= 0 {
				names := strings.Split(action[:assignment], ",")
				if len(names) != 2 || !strings.HasPrefix(strings.TrimSpace(names[0]), "$") || !strings.HasPrefix(strings.TrimSpace(names[1]), "$") {
					return nil, "", fmt.Errorf("invalid range assignment")
				}
				node.rangeIndex, node.rangeElement = strings.TrimSpace(names[0]), strings.TrimSpace(names[1])
				action = strings.TrimSpace(action[assignment+2:])
			}
		}
		expr, err := compileExpression(action)
		if err != nil {
			return nil, "", err
		}
		node.expr = expr
		if node.kind == "if" || node.kind == "range" {
			var stop string
			node.children, stop, err = parseTemplate(source, position, depth+1)
			if err != nil {
				return nil, "", err
			}
			if stop == "else" {
				node.otherwise, stop, err = parseTemplate(source, position, depth+1)
				if err != nil {
					return nil, "", err
				}
			}
			if stop != "end" {
				return nil, "", fmt.Errorf("unclosed template %s", node.kind)
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, "", nil
}

func tokenizeExpression(source string) ([]templateToken, error) {
	var out []templateToken
	for i := 0; i < len(source); {
		if unicode.IsSpace(rune(source[i])) {
			i++
			continue
		}
		if source[i] == '(' || source[i] == ')' {
			out = append(out, templateToken{text: source[i : i+1]})
			i++
			continue
		}
		if source[i] == '"' {
			i++
			var text strings.Builder
			closed := false
			for i < len(source) {
				if source[i] == '"' {
					i++
					closed = true
					break
				}
				if source[i] == '\\' && i+1 < len(source) && (source[i+1] == '"' || source[i+1] == '\\') {
					i++
				}
				text.WriteByte(source[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unclosed template string")
			}
			out = append(out, templateToken{text: text.String(), literal: true})
			continue
		}
		start := i
		for i < len(source) && !unicode.IsSpace(rune(source[i])) && source[i] != '(' && source[i] != ')' {
			i++
		}
		out = append(out, templateToken{text: source[start:i]})
	}
	return out, nil
}

func compileExpression(source string) (expression, error) {
	tokens, err := tokenizeExpression(source)
	if err != nil {
		return expression{}, err
	}
	pos := 0
	expr, err := parseExpression(tokens, &pos, 0)
	if err == nil && pos != len(tokens) {
		err = fmt.Errorf("unexpected template expression token")
	}
	return expr, err
}

func parseExpression(tokens []templateToken, position *int, depth int) (expression, error) {
	if depth > 32 || *position >= len(tokens) {
		return expression{}, fmt.Errorf("invalid template expression")
	}
	token := tokens[*position]
	*position++
	if token.literal {
		return expression{kind: "literal", text: token.text}, nil
	}
	if token.text == "(" {
		expr, err := parseExpression(tokens, position, depth+1)
		if err != nil {
			return expr, err
		}
		if *position >= len(tokens) || tokens[*position].text != ")" {
			return expression{}, fmt.Errorf("unclosed expression group")
		}
		*position++
		return expr, nil
	}
	if strings.HasPrefix(token.text, ".") || strings.HasPrefix(token.text, "$") {
		return expression{kind: "variable", text: token.text}, nil
	}
	if token.text == "true" || token.text == "false" {
		value := "True"
		if token.text == "false" {
			value = ""
		}
		return expression{kind: "literal", text: value}, nil
	}
	if _, err := strconv.ParseFloat(token.text, 64); err == nil {
		return expression{kind: "literal", text: token.text}, nil
	}
	minimum, maximum := 0, 0
	switch token.text {
	case "and", "or":
		minimum, maximum = 2, 100
	case "eq", "ne":
		minimum, maximum = 2, 2
	case "join":
		minimum, maximum = 2, 2
	case "re_replace":
		minimum, maximum = 3, 3
	default:
		return expression{}, fmt.Errorf("unsupported template function %q", token.text)
	}
	expr := expression{kind: "call", text: token.text}
	for *position < len(tokens) && tokens[*position].text != ")" && len(expr.args) < maximum {
		arg, err := parseExpression(tokens, position, depth+1)
		if err != nil {
			return expression{}, err
		}
		expr.args = append(expr.args, arg)
	}
	if len(expr.args) < minimum {
		return expression{}, fmt.Errorf("template function %s requires %d arguments", token.text, minimum)
	}
	return expr, nil
}

func (t *Template) Render(ctx context.Context, variables Variables) (string, error) {
	return t.RenderWithModifier(ctx, variables, nil)
}

func (t *Template) RenderWithModifier(ctx context.Context, variables Variables, modifier func(string) string) (string, error) {
	var out strings.Builder
	steps := MaxTemplateSteps
	err := renderNodes(ctx, t.nodes, variables, &out, &steps, modifier)
	return out.String(), err
}

func renderNodes(ctx context.Context, nodes []templateNode, variables Variables, out *strings.Builder, steps *int, modifier func(string) string) error {
	for _, node := range nodes {
		*steps--
		if *steps < 0 {
			return fmt.Errorf("template execution budget exceeded")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if node.kind == "text" {
			out.WriteString(node.text)
		} else {
			value, err := node.expr.evaluate(variables)
			if err != nil {
				return err
			}
			switch node.kind {
			case "value":
				text := value.Text
				if modifier != nil {
					text = modifier(text)
				}
				out.WriteString(text)
			case "if":
				branch := node.otherwise
				if truthy(value) {
					branch = node.children
				}
				if err := renderNodes(ctx, branch, variables, out, steps, modifier); err != nil {
					return err
				}
			case "range":
				if len(value.Items) == 0 {
					if err := renderNodes(ctx, node.otherwise, variables, out, steps, modifier); err != nil {
						return err
					}
				}
				for i, item := range value.Items {
					local := variables.Clone()
					local["."] = TemplateValue{Text: item}
					if node.rangeIndex != "" {
						local[node.rangeIndex] = TemplateValue{Text: strconv.Itoa(i)}
						local[node.rangeElement] = TemplateValue{Text: item}
					}
					if err := renderNodes(ctx, node.children, local, out, steps, modifier); err != nil {
						return err
					}
				}
			}
		}
		if out.Len() > MaxTemplateBytes {
			return fmt.Errorf("template output exceeds byte budget")
		}
	}
	return nil
}

func truthy(value TemplateValue) bool { return value.Text != "" || len(value.Items) > 0 }

func (e expression) evaluate(variables Variables) (TemplateValue, error) {
	if e.kind == "literal" {
		return TemplateValue{Text: e.text}, nil
	}
	if e.kind == "variable" {
		return variables[e.text], nil
	}
	args := make([]TemplateValue, 0, len(e.args))
	for _, arg := range e.args {
		value, err := arg.evaluate(variables)
		if err != nil {
			return TemplateValue{}, err
		}
		args = append(args, value)
		if e.text == "or" && truthy(value) {
			return value, nil
		}
		if e.text == "and" && !truthy(value) {
			return value, nil
		}
	}
	switch e.text {
	case "and", "or":
		return args[len(args)-1], nil
	case "eq", "ne":
		same := args[0].Text == args[1].Text
		if same == (e.text == "eq") {
			return TemplateValue{Text: "True"}, nil
		}
		return TemplateValue{}, nil
	case "join":
		return TemplateValue{Text: strings.Join(args[0].Items, args[1].Text)}, nil
	case "re_replace":
		re, err := boundedRegex(args[1].Text)
		if err != nil {
			return TemplateValue{}, err
		}
		text, err := re.Replace(args[0].Text, args[2].Text, -1, -1)
		return TemplateValue{Text: text}, err
	}
	return TemplateValue{}, fmt.Errorf("unsupported expression")
}

func boundedRegex(pattern string) (*regexp2.Regexp, error) {
	if len(pattern) > 16384 {
		return nil, fmt.Errorf("regular expression exceeds byte budget")
	}
	re, err := regexp2.Compile(normalizeRegexBlocks(pattern), regexp2.None)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}
	re.MatchTimeout = 100 * time.Millisecond
	return re, nil
}

// regexp2 follows .NET matching, but omits .NET's Unicode block aliases.
// IsCyrillic denotes the block U+0400..U+04FF, not the broader Unicode script.
// IsCJKUnifiedIdeographs similarly denotes U+4E00..U+9FFF. Boundaries follow
// dotnet/runtime v9.0.0 RegexCharClass.cs, not Unicode script membership.
func normalizeRegexBlocks(pattern string) string {
	var out strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			matched := false
			for _, property := range []string{`\p{IsCyrillic}`, `\P{IsCyrillic}`, `\p{IsCJKUnifiedIdeographs}`, `\P{IsCJKUnifiedIdeographs}`} {
				if strings.HasPrefix(pattern[i:], property) {
					rangeText, complement := `\u0400-\u04FF`, `\u0000-\u03FF\u0500-\uFFFF`
					if property == `\p{IsCJKUnifiedIdeographs}` || property == `\P{IsCJKUnifiedIdeographs}` {
						rangeText, complement = `\u4E00-\u9FFF`, `\u0000-\u4DFF\uA000-\uFFFF`
					}
					if property[1] == 'p' {
						if !inClass {
							out.WriteByte('[')
						}
						out.WriteString(rangeText)
						if !inClass {
							out.WriteByte(']')
						}
					} else {
						if inClass {
							out.WriteString(complement)
						} else {
							out.WriteString(`[^` + rangeText + `]`)
						}
					}
					i += len(property) - 1
					matched = true
					break
				}
			}
			if matched {
				continue
			}
			out.WriteByte(pattern[i])
			i++
			out.WriteByte(pattern[i])
			continue
		}
		if pattern[i] == '[' {
			inClass = true
		}
		if pattern[i] == ']' {
			inClass = false
		}
		out.WriteByte(pattern[i])
	}
	return out.String()
}

func renderText(ctx context.Context, text string, variables Variables) (string, error) {
	t, err := CompileTemplate(text)
	if err != nil {
		return "", err
	}
	return t.Render(ctx, variables)
}
