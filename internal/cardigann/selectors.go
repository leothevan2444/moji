package cardigann

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/ohler55/ojg/jp"
)

func selectHTML(root *goquery.Selection, selector string, all bool) (*goquery.Selection, error) {
	if strings.HasPrefix(selector, ":root") {
		selector = strings.TrimSpace(strings.TrimPrefix(selector, ":root"))
		for root.Parent().Length() > 0 {
			root = root.Parent()
		}
		if selector == "" {
			return root, nil
		}
	}
	plan, err := compileCSS(selector)
	if err != nil {
		return nil, fmt.Errorf("invalid HTML selector: %w", err)
	}
	return plan.query(root, all)
}

func HTMLField(ctx context.Context, root *goquery.Selection, selector Selector, fc FilterContext) (string, bool, error) {
	if selector.Text != nil {
		text, err := renderText(ctx, string(*selector.Text), fc.Variables)
		if err != nil {
			return "", false, err
		}
		text, err = ApplyFilters(ctx, text, selector.Filters, fc)
		return text, true, err
	}
	selection := root
	if selector.Selector != "" {
		query, err := renderText(ctx, selector.Selector, fc.Variables)
		if err != nil {
			return "", false, err
		}
		selection, err = selectHTML(root, query, false)
		if err != nil {
			if selector.Optional {
				return optionalField(ctx, selector, fc)
			}
			return "", false, err
		}
		if selection.Length() == 0 {
			return optionalField(ctx, selector, fc)
		}
	}
	if selector.Remove != "" {
		query, err := renderText(ctx, selector.Remove, fc.Variables)
		if err != nil {
			return "", false, err
		}
		matcher, err := cascadia.Compile(query)
		if err != nil {
			return "", false, fmt.Errorf("invalid remove selector: %w", err)
		}
		selection.FindMatcher(matcher).Remove()
	}
	var text string
	if len(selector.Case) > 0 {
		matched := false
		for _, c := range selector.Case {
			query, err := renderText(ctx, c.Selector, fc.Variables)
			if err != nil {
				return "", false, err
			}
			result, err := selectHTML(selection, query, false)
			if err != nil {
				return "", false, err
			}
			if result.Length() > 0 {
				text, err = renderText(ctx, string(c.Value), fc.Variables)
				if err != nil {
					return "", false, err
				}
				matched = true
				break
			}
		}
		if !matched {
			return optionalField(ctx, selector, fc)
		}
	} else if selector.Attribute != "" {
		var found bool
		text, found = selection.Attr(selector.Attribute)
		if !found {
			return optionalField(ctx, selector, fc)
		}
	} else {
		text = selection.Text()
	}
	text, err := ApplyFilters(ctx, strings.TrimSpace(text), selector.Filters, fc)
	if err != nil {
		return "", false, err
	}
	if text == "" && selector.Optional && selector.Default != nil {
		return optionalField(ctx, selector, fc)
	}
	return text, true, nil
}

func optionalField(ctx context.Context, selector Selector, fc FilterContext) (string, bool, error) {
	if !selector.Optional {
		return "", false, fmt.Errorf("required selector or attribute did not match")
	}
	if selector.Default != nil {
		text, err := renderText(ctx, string(*selector.Default), fc.Variables)
		return text, true, err
	}
	return "", false, nil
}

type jsonCondition struct {
	name     string
	argument string
}
type jsonSelector struct {
	path       jp.Expr
	root       bool
	conditions []jsonCondition
}

func parseJSONSelector(source string) (jsonSelector, error) {
	var result jsonSelector
	cut, brackets, quote := len(source), 0, byte(0)
	for i := 0; i < len(source); i++ {
		c := source[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == '[' {
			brackets++
		}
		if c == ']' {
			brackets--
		}
		if c == ':' && brackets == 0 {
			cut = i
			break
		}
	}
	path := strings.TrimLeft(strings.TrimSpace(source[:cut]), ".")
	if path == "" || path == "$" {
		result.root = true
	} else {
		var err error
		result.path, err = jp.ParseString(path)
		if err != nil {
			return result, fmt.Errorf("invalid JSON path: %w", err)
		}
	}
	for pos := cut; pos < len(source); {
		if source[pos] != ':' {
			return result, fmt.Errorf("invalid JSON selector condition")
		}
		pos++
		start := pos
		for pos < len(source) && source[pos] != '(' {
			pos++
		}
		if pos == len(source) {
			return result, fmt.Errorf("unclosed JSON selector condition")
		}
		name := source[start:pos]
		if name != "has" && name != "not" && name != "contains" {
			return result, fmt.Errorf("unsupported JSON condition %q", name)
		}
		pos++
		start = pos
		depth := 1
		quote = 0
		for pos < len(source) && depth > 0 {
			c := source[pos]
			if quote != 0 {
				if c == '\\' {
					pos++
				} else if c == quote {
					quote = 0
				}
			} else {
				switch c {
				case '"', '\'':
					quote = c
				case '(':
					depth++
				case ')':
					depth--
				}
			}
			pos++
		}
		if depth != 0 {
			return result, fmt.Errorf("unclosed JSON condition")
		}
		argument := source[start : pos-1]
		if name != "contains" {
			if _, err := parseJSONSelector(argument); err != nil {
				return result, err
			}
		}
		result.conditions = append(result.conditions, jsonCondition{name: name, argument: argument})
	}
	return result, nil
}

func (s jsonSelector) selectValues(value any) ([]any, error) {
	values := []any{value}
	if !s.root {
		values = s.path.Get(value)
	}
	var result []any
	for _, item := range values {
		matches := true
		for _, condition := range s.conditions {
			if condition.name == "contains" {
				if !strings.Contains(jsonScalarText(item), condition.argument) {
					matches = false
					break
				}
				continue
			}
			child, err := parseJSONSelector(condition.argument)
			if err != nil {
				return nil, err
			}
			selected, err := child.selectValues(item)
			if err != nil {
				return nil, err
			}
			if (len(selected) > 0) != (condition.name == "has") {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, item)
		}
	}
	return result, nil
}

func JSONField(ctx context.Context, root any, selector Selector, fc FilterContext) (string, bool, error) {
	if selector.Text != nil {
		text, err := renderText(ctx, string(*selector.Text), fc.Variables)
		if err != nil {
			return "", false, err
		}
		text, err = ApplyFilters(ctx, text, selector.Filters, fc)
		return text, true, err
	}
	var value any
	if selector.Selector != "" {
		query, err := renderText(ctx, selector.Selector, fc.Variables)
		if err != nil {
			return "", false, err
		}
		compiled, err := parseJSONSelector(query)
		if err != nil {
			return "", false, err
		}
		selected, err := compiled.selectValues(root)
		if err != nil {
			return "", false, err
		}
		if len(selected) == 0 {
			return optionalField(ctx, selector, fc)
		}
		value = selected[0]
	}
	text := jsonScalarText(value)
	if items, ok := value.([]any); ok {
		parts := make([]string, 0, len(items))
		for _, item := range items {
			parts = append(parts, jsonScalarText(item))
		}
		text = strings.Join(parts, ",")
	}
	if len(selector.Case) > 0 {
		for _, c := range selector.Case {
			if c.Selector == "*" || value != nil && text == c.Selector {
				var err error
				text, err = renderText(ctx, string(c.Value), fc.Variables)
				if err != nil {
					return "", false, err
				}
				value = text
				break
			}
		}
	}
	if value == nil {
		return optionalField(ctx, selector, fc)
	}
	if selector.Attribute != "" {
		attribute, err := parseJSONSelector(selector.Attribute)
		if err != nil {
			return "", false, err
		}
		values, err := attribute.selectValues(value)
		if err != nil {
			return "", false, err
		}
		if len(values) == 0 {
			return optionalField(ctx, selector, fc)
		}
		text = jsonScalarText(values[0])
	}
	text, err := ApplyFilters(ctx, strings.TrimSpace(text), selector.Filters, fc)
	if err != nil {
		return "", false, err
	}
	if text == "" && selector.Optional && selector.Default != nil {
		return optionalField(ctx, selector, fc)
	}
	return text, true, nil
}
