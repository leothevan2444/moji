package cardigann

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

const scopeAttribute = "data-moji-cardigann-scope"

type cssRelation struct {
	attribute string
	selector  *cssPlan
}
type cssPlan struct {
	matcher   cascadia.Selector
	relations []cssRelation
	scoped    bool
}

// CSS extensions bridge AngleSharp's :scope, unquoted :contains arguments and
// relative :has combinators without rewriting the downloaded rule or leaking
// temporary scope attributes into extracted fields.
func compileCSS(source string) (*cssPlan, error) { return compileCSSDepth(source, 0) }

func compileCSSDepth(source string, depth int) (*cssPlan, error) {
	if depth > 16 {
		return nil, fmt.Errorf("CSS nesting budget exceeded")
	}
	plan := &cssPlan{}
	var out strings.Builder
	for i := 0; i < len(source); {
		if source[i] == '"' || source[i] == '\'' {
			quote := source[i]
			out.WriteByte(quote)
			i++
			for i < len(source) {
				c := source[i]
				out.WriteByte(c)
				i++
				if c == '\\' && i < len(source) {
					out.WriteByte(source[i])
					i++
					continue
				}
				if c == quote {
					break
				}
			}
			continue
		}
		if strings.HasPrefix(source[i:], ":scope") && (i+6 == len(source) || !isCSSIdentifier(source[i+6])) {
			out.WriteString("[" + scopeAttribute + "]")
			plan.scoped = true
			i += 6
			continue
		}
		prefix := ""
		for _, candidate := range []string{":contains(", ":containsown(", ":has("} {
			if strings.HasPrefix(source[i:], candidate) {
				prefix = candidate
				break
			}
		}
		if prefix != "" {
			start := i + len(prefix)
			end, err := cssClosingParen(source, start)
			if err != nil {
				return nil, err
			}
			argument := strings.TrimSpace(source[start:end])
			if prefix == ":has(" && argument != "" && strings.ContainsRune(">+~", rune(argument[0])) {
				child, err := compileCSSDepth(":scope "+argument, depth+1)
				if err != nil {
					return nil, err
				}
				attribute := "data-moji-cardigann-relative-" + strconv.Itoa(len(plan.relations))
				plan.relations = append(plan.relations, cssRelation{attribute: attribute, selector: child})
				out.WriteString("[" + attribute + "]")
				i = end + 1
				continue
			}
			if prefix != ":has(" && argument != "" && argument[0] != '"' && argument[0] != '\'' {
				out.WriteString(prefix + strconv.Quote(argument) + ")")
				i = end + 1
				continue
			}
		}
		out.WriteByte(source[i])
		i++
	}
	matcher, err := cascadia.Compile(out.String())
	if err != nil {
		return nil, err
	}
	plan.matcher = matcher
	return plan, nil
}

func isCSSIdentifier(c byte) bool {
	return c == '-' || c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func cssClosingParen(source string, start int) (int, error) {
	depth := 1
	quote := byte(0)
	for i := start; i < len(source); i++ {
		c := source[i]
		if c == '\\' {
			i++
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == '(' {
			depth++
		}
		if c == ')' {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unclosed CSS function")
}

func (p *cssPlan) query(root *goquery.Selection, all bool) (*goquery.Selection, error) {
	budget := 1000000
	return p.queryBounded(root, all, &budget)
}

func (p *cssPlan) queryBounded(root *goquery.Selection, all bool, budget *int) (*goquery.Selection, error) {
	if !p.scoped && len(p.relations) == 0 {
		return queryCSS(root, p.matcher, all), nil
	}
	document := root
	for document.Parent().Length() > 0 {
		document = document.Parent()
	}
	var nodes []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			nodes = append(nodes, n)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, n := range document.Nodes {
		walk(n)
	}
	if len(nodes) > 50000 {
		return nil, fmt.Errorf("CSS scope node budget exceeded")
	}
	*budget -= len(nodes)
	if *budget < 0 {
		return nil, fmt.Errorf("CSS relational evaluation budget exceeded")
	}
	// Only nodes modified by this invocation are saved. Existing source
	// attributes with the reserved names are restored after matching.
	saved := map[*html.Node][]html.Attribute{}
	mutate := func(n *html.Node, key string, present bool) {
		if _, ok := saved[n]; !ok {
			saved[n] = append([]html.Attribute(nil), n.Attr...)
		}
		attrs := n.Attr[:0]
		for _, a := range n.Attr {
			if a.Key != key {
				attrs = append(attrs, a)
			}
		}
		n.Attr = attrs
		if present {
			n.Attr = append(n.Attr, html.Attribute{Key: key, Val: ""})
		}
	}
	defer func() {
		for n, attributes := range saved {
			n.Attr = attributes
		}
	}()
	for _, n := range nodes {
		for _, a := range append([]html.Attribute(nil), n.Attr...) {
			if a.Key == scopeAttribute || strings.HasPrefix(a.Key, "data-moji-cardigann-relative-") {
				mutate(n, a.Key, false)
			}
		}
	}
	if p.scoped {
		for _, n := range root.Nodes {
			mutate(n, scopeAttribute, true)
		}
	}
	for _, relation := range p.relations {
		for _, n := range nodes {
			selected, err := relation.selector.queryBounded(goquery.NewDocumentFromNode(n).Selection, true, budget)
			if err != nil {
				return nil, err
			}
			if selected.Length() > 0 {
				mutate(n, relation.attribute, true)
			}
		}
	}
	if p.scoped {
		return queryCSS(document, p.matcher, all), nil
	}
	return queryCSS(root, p.matcher, all), nil
}

func queryCSS(root *goquery.Selection, matcher cascadia.Selector, all bool) *goquery.Selection {
	var result *goquery.Selection
	if root.IsMatcher(matcher) {
		result = root.FilterMatcher(matcher)
	} else {
		result = root.FindMatcher(matcher)
	}
	if !all {
		result = result.First()
	}
	return result
}
