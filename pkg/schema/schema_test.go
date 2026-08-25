package schema

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func f64(v float64) *float64 { return &v }

func TestGeneratedSchemaValidatesRealEnvelope(t *testing.T) {
	raw, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	sch, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", sch); err != nil {
		t.Fatal(err)
	}
	compiled, err := c.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}

	// Validate what the app actually publishes: an Assembler-produced
	// snapshot (which normalizes collections to empty-not-null).
	ref := snapshot.LocationRef{Label: "X", Zip: "92057", Lat: 33.24, Lon: -117.29}
	asm := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{"nws", "coops"})
	asm.Apply(snapshot.Fragment{Provider: "nws", Kind: snapshot.KindObs, FetchedAt: time.Now(),
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{
			snapshot.Key(ref): {Current: &snapshot.Conditions{Temp: f64(21.5)}},
		}})
	// A marine block with nil collections (REVIEW M1): the per-provider copy
	// must publish arrays too, or the live --json fails this very schema.
	asm.Apply(snapshot.Fragment{Provider: "coops", Kind: snapshot.KindMarine, FetchedAt: time.Now(),
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{
			snapshot.Key(ref): {Marine: &snapshot.Marine{}},
		}})
	envBytes, err := json.Marshal(asm.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(envBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Fatalf("real envelope fails its own schema: %v", err)
	}
}

func TestSchemaRejectsUnknownFields(t *testing.T) {
	raw, _ := Generate()
	sch, _ := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	c := jsonschema.NewCompiler()
	_ = c.AddResource("schema.json", sch)
	compiled, err := c.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	bad := map[string]any{"schema_version": "1.0.0-rc", "generated_at": "2026-08-23T00:00:00Z",
		"locations": []any{}, "providers": []any{}, "warnings": []any{}, "smuggled": true}
	if err := compiled.Validate(bad); err == nil {
		t.Fatal("additionalProperties:false must reject unknown envelope fields (PLAN #8)")
	}
}

func TestSchemaCarriesRCVersion(t *testing.T) {
	raw, _ := Generate()
	if !bytes.Contains(raw, []byte("1.0.0-rc")) {
		t.Fatal("schema $id must carry the -rc version until B5 ratifies (§10.3)")
	}
}

func TestPublishedSchemaMatchesGenerator(t *testing.T) {
	// REVIEW M2: the checked-in schema is what $id points at; it must be
	// byte-identical to Generate() (regenerate with `make schema`).
	want, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("watchpost-report.v" + snapshot.SchemaVersion + ".schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatal("pkg/schema/*.schema.json is stale — run `make schema`")
	}
}
