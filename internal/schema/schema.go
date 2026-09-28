// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package schema validates a JSON document against the subset of JSON Schema
// 2020-12 that passmcp-reporting's generated schemas use: $ref into $defs,
// type, properties, required, enum, const and items.
//
// It exists so an export can be checked against the published graph schema
// without a dependency. A schema that uses any other keyword is refused, not
// half-checked: a validator that silently ignores a rule is worse than none.
package schema

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// known is every keyword this validator implements, plus the annotations it
// may safely ignore.
var known = map[string]bool{
	"$schema": true, "$id": true, "$comment": true, "$defs": true, "$ref": true,
	"title": true, "description": true,
	"type": true, "properties": true, "required": true, "enum": true, "const": true, "items": true,
}

// Validator checks documents against one schema.
type Validator struct {
	root map[string]any
}

// New parses schemaJSON and refuses a schema that uses a keyword the
// validator does not implement.
func New(schemaJSON []byte) (*Validator, error) {
	var root map[string]any
	if err := json.Unmarshal(schemaJSON, &root); err != nil {
		return nil, fmt.Errorf("schema is not JSON: %w", err)
	}
	if err := checkKeywords(root, "#"); err != nil {
		return nil, err
	}
	return &Validator{root: root}, nil
}

// checkKeywords walks every schema object and refuses unknown keywords.
func checkKeywords(s map[string]any, at string) error {
	for k, v := range s {
		if !known[k] {
			return fmt.Errorf("%s: keyword %q is not supported", at, k)
		}
		if err := checkChildren(k, v, at); err != nil {
			return err
		}
	}
	return nil
}

// checkChildren descends into the keywords whose values are schemas.
func checkChildren(k string, v any, at string) error {
	switch k {
	case "$defs", "properties":
		m, _ := v.(map[string]any)
		for name, sub := range m {
			if sm, ok := sub.(map[string]any); ok {
				if err := checkKeywords(sm, at+"/"+k+"/"+name); err != nil {
					return err
				}
			}
		}
	case "items":
		if sm, ok := v.(map[string]any); ok {
			return checkKeywords(sm, at+"/items")
		}
	}
	return nil
}

// Validate checks doc and returns every violation, sorted, each with its
// JSON path. An empty result means the document is valid.
func (v *Validator) Validate(doc []byte) ([]string, error) {
	var value any
	if err := json.Unmarshal(doc, &value); err != nil {
		return nil, fmt.Errorf("document is not JSON: %w", err)
	}
	var problems []string
	v.check(v.root, value, "$", &problems)
	sort.Strings(problems)
	return problems, nil
}

func (v *Validator) check(s map[string]any, value any, path string, out *[]string) {
	if ref, ok := s["$ref"].(string); ok {
		target, err := v.resolve(ref)
		if err != nil {
			*out = append(*out, path+": "+err.Error())
			return
		}
		v.check(target, value, path, out)
	}
	if t, ok := s["type"].(string); ok && !isType(value, t) {
		*out = append(*out, fmt.Sprintf("%s: want %s", path, t))
		return
	}
	checkEnumConst(s, value, path, out)
	switch val := value.(type) {
	case map[string]any:
		v.checkObject(s, val, path, out)
	case []any:
		if items, ok := s["items"].(map[string]any); ok {
			for i, item := range val {
				v.check(items, item, fmt.Sprintf("%s[%d]", path, i), out)
			}
		}
	}
}

func (v *Validator) checkObject(s map[string]any, obj map[string]any, path string, out *[]string) {
	req, _ := s["required"].([]any)
	for _, r := range req {
		name, _ := r.(string)
		if _, ok := obj[name]; !ok {
			*out = append(*out, fmt.Sprintf("%s: missing required %q", path, name))
		}
	}
	props, _ := s["properties"].(map[string]any)
	for name, sub := range props {
		val, ok := obj[name]
		sm, isSchema := sub.(map[string]any)
		if ok && isSchema {
			v.check(sm, val, path+"."+name, out)
		}
	}
}

func checkEnumConst(s map[string]any, value any, path string, out *[]string) {
	if c, ok := s["const"]; ok && !equal(c, value) {
		*out = append(*out, fmt.Sprintf("%s: want %v", path, c))
	}
	enum, ok := s["enum"].([]any)
	if !ok {
		return
	}
	for _, e := range enum {
		if equal(e, value) {
			return
		}
	}
	*out = append(*out, fmt.Sprintf("%s: %v is not one of %v", path, value, enum))
}

func (v *Validator) resolve(ref string) (map[string]any, error) {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, fmt.Errorf("unsupported $ref %q", ref)
	}
	defs, _ := v.root["$defs"].(map[string]any)
	target, ok := defs[strings.TrimPrefix(ref, prefix)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unresolved $ref %q", ref)
	}
	return target, nil
}

func isType(value any, t string) bool {
	switch t {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		f, ok := value.(float64)
		return ok && f == math.Trunc(f)
	case "null":
		return value == nil
	}
	return false
}

func equal(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(ja) == string(jb)
}
