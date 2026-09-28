// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package schema

import (
	"strings"
	"testing"

	"satellion.com/passmcp-reporting/spec"
)

const tiny = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$ref": "#/$defs/Doc",
  "$defs": {
    "Doc": {"type": "object", "required": ["kind", "items"],
      "properties": {
        "kind": {"type": "string", "enum": ["a", "b"]},
        "v": {"const": "x"},
        "n": {"type": "integer"},
        "f": {"type": "number"},
        "ok": {"type": "boolean"},
        "items": {"type": "array", "items": {"$ref": "#/$defs/Item"}}
      }},
    "Item": {"type": "object", "required": ["id"], "properties": {"id": {"type": "string"}}}
  }
}`

func TestValidDocumentHasNoProblems(t *testing.T) {
	v, err := New([]byte(tiny))
	if err != nil {
		t.Fatal(err)
	}
	p, err := v.Validate([]byte(`{"kind":"a","v":"x","n":3,"f":1.5,"ok":true,"items":[{"id":"1"}]}`))
	if err != nil || len(p) != 0 {
		t.Fatalf("got %v %v", p, err)
	}
}

func TestEveryViolationIsReportedWithItsPath(t *testing.T) {
	v, _ := New([]byte(tiny))
	p, err := v.Validate([]byte(`{"kind":"c","v":"y","n":1.5,"f":"no","ok":1,"items":[{}, 3]}`))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(p, "\n")
	for _, want := range []string{`$.kind: c is not one of`, `$.v: want x`, `$.n: want integer`, `$.f: want number`,
		`$.ok: want boolean`, `$.items[0]: missing required "id"`, `$.items[1]: want object`} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
	if p, _ := v.Validate([]byte(`{}`)); len(p) != 2 {
		t.Errorf("want both required fields missing, got %v", p)
	}
}

func TestSchemaWithAnUnsupportedKeywordIsRefused(t *testing.T) {
	for _, s := range []string{
		`{"$defs":{"A":{"pattern":"x"}}}`,
		`{"properties":{"a":{"minimum":1}}}`,
		`{"items":{"oneOf":[]}}`,
	} {
		if _, err := New([]byte(s)); err == nil {
			t.Errorf("%s: want refusal", s)
		}
	}
	if _, err := New([]byte(`not json`)); err == nil {
		t.Error("want error for non-JSON schema")
	}
}

func TestBadReferencesAndDocuments(t *testing.T) {
	v, _ := New([]byte(`{"$ref":"#/$defs/Missing","$defs":{}}`))
	if p, _ := v.Validate([]byte(`{}`)); len(p) != 1 || !strings.Contains(p[0], "unresolved") {
		t.Errorf("got %v", p)
	}
	v, _ = New([]byte(`{"$ref":"http://elsewhere"}`))
	if p, _ := v.Validate([]byte(`{}`)); len(p) != 1 || !strings.Contains(p[0], "unsupported") {
		t.Errorf("got %v", p)
	}
	if _, err := v.Validate([]byte(`{`)); err == nil {
		t.Error("want error for non-JSON document")
	}
	v, _ = New([]byte(`{"type":"null"}`))
	if p, _ := v.Validate([]byte(`null`)); len(p) != 0 {
		t.Errorf("null: %v", p)
	}
	v, _ = New([]byte(`{"type":"mystery"}`))
	if p, _ := v.Validate([]byte(`1`)); len(p) != 1 {
		t.Errorf("unknown type must fail: %v", p)
	}
}

// The published graph schema uses only what the validator implements.
func TestThePublishedGraphSchemaIsSupported(t *testing.T) {
	if _, err := New(spec.GraphSchema); err != nil {
		t.Fatal(err)
	}
}
