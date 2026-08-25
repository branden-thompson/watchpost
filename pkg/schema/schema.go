// Package schema generates the published JSON Schema (draft 2020-12) for the
// watchpost report envelope, by reflection over platform/snapshot types — the
// struct IS the source of truth, so schema drift is impossible by
// construction (M5; architecture §10.3). additionalProperties is false on
// every object (PLAN red-team #8). Pointer fields are nullable
// (["<type>","null"] — the null-parity rule).
package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// ID is the schema's $id: the raw path of the checked-in file, which
// TestPublishedSchemaMatchesGenerator keeps byte-identical to Generate()
// (REVIEW M2). The -rc suffix rides SchemaVersion until ratified.
const ID = "https://raw.githubusercontent.com/branden-thompson/watchpost/main/pkg/schema/watchpost-report.v" // + version

// Generate returns the JSON Schema document for snapshot.Snapshot.
func Generate() ([]byte, error) {
	root := typeSchema(reflect.TypeOf(snapshot.Snapshot{}), false)
	doc := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         ID + snapshot.SchemaVersion + ".schema.json",
		"title":       "watchpost report",
		"description": "Machine-readable weather report envelope. Harmonization never blends: NWS wins outright; secondaries fill nulls only (fill_from records provenance).",
	}
	for k, v := range root {
		doc[k] = v
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot encode schema: %w", err)
	}
	return out, nil
}

var timeType = reflect.TypeOf(time.Time{})

// maxDepth bounds the type recursion: the snapshot type graph is ~5 deep; a
// depth beyond 12 means an accidentally-recursive type, which must fail the
// generator loudly rather than loop (bounded recursion — P10-02 discipline).
const maxDepth = 12

// typeSchema maps a Go type to its schema node. nullable wraps the type union
// with "null" (pointer fields).
func typeSchema(t reflect.Type, nullable bool) map[string]any {
	return typeSchemaDepth(t, nullable, 0)
}

func typeSchemaDepth(t reflect.Type, nullable bool, depth int) map[string]any {
	if err := invariant.Check(depth <= maxDepth, "schema generator exceeded type depth — recursive contract type?"); err != nil {
		return map[string]any{"$comment": err.Error()}
	}
	if t == timeType {
		return withNull(map[string]any{"type": "string", "format": "date-time"}, nullable)
	}
	switch t.Kind() {
	case reflect.Pointer:
		return typeSchemaDepth(t.Elem(), true, depth+1)
	case reflect.String:
		return withNull(map[string]any{"type": "string"}, nullable)
	case reflect.Float64, reflect.Float32:
		return withNull(map[string]any{"type": "number"}, nullable)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return withNull(map[string]any{"type": "integer"}, nullable)
	case reflect.Bool:
		return withNull(map[string]any{"type": "boolean"}, nullable)
	case reflect.Slice:
		return withNull(map[string]any{"type": "array", "items": typeSchemaDepth(t.Elem(), false, depth+1)}, nullable)
	case reflect.Map:
		return withNull(map[string]any{"type": "object", "additionalProperties": typeSchemaDepth(t.Elem(), false, depth+1)}, nullable)
	case reflect.Struct:
		props := map[string]any{}
		var required []string
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, opts, _ := strings.Cut(tag, ",")
			if err := invariant.Check(name != "", "schema: exported field without json name: "+t.Name()+"."+f.Name); err != nil {
				continue // reflection gate in snapshot tests catches this loudly
			}
			node := typeSchemaDepth(f.Type, false, depth+1)
			if e := f.Tag.Get("enum"); e != "" { // closed value sets documented on the contract struct (REVIEW M3)
				node["enum"] = strings.Split(e, ",")
			}
			props[name] = node
			if !strings.Contains(opts, "omitempty") {
				required = append(required, name)
			}
		}
		node := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			node["required"] = required
		}
		return withNull(node, nullable)
	default:
		return map[string]any{} // permissive for unknown kinds; reflection gate guards
	}
}

func withNull(node map[string]any, nullable bool) map[string]any {
	if !nullable {
		return node
	}
	if ty, ok := node["type"].(string); ok {
		node["type"] = []string{ty, "null"}
	}
	return node
}
