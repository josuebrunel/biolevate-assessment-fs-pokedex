package data

import (
	"strings"
	"testing"
)

func TestEmbedded(t *testing.T) {
	ds, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if got := [3]int{len(ds.Pokemon), len(ds.Moves), len(ds.Abilities)}; got != [3]int{151, 164, 114} {
		t.Errorf("counts (pokemon, moves, abilities) = %v, want [151 164 114]", got)
	}
}

func TestParseValidation(t *testing.T) {
	const stats = `"stats":{"hp":1,"attack":1,"defense":1,"special-attack":1,"special-defense":1,"speed":1}`
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			"valid",
			`{"moves":[{"id":1,"name":"m","type":"normal","damage_class":"status","generation":"g"}],
			  "abilities":[{"id":1,"name":"a","generation":"g"}],
			  "pokemon":[{"id":1,"name":"p","types":["normal"],` + stats + `,"abilities":["a"],"moves":["m"],"species":{"generation":"g"}}]}`,
			"",
		},
		{
			"unknown move",
			`{"pokemon":[{"id":1,"name":"p","types":["normal"],` + stats + `,"moves":["ghost"],"species":{"generation":"g"}}]}`,
			`unknown move "ghost"`,
		},
		{
			"missing stat",
			`{"pokemon":[{"id":1,"name":"p","types":["normal"],"stats":{"hp":1},"species":{"generation":"g"}}]}`,
			"missing stat",
		},
		{
			"too many types",
			`{"pokemon":[{"id":1,"name":"p","types":["a","b","c"],` + stats + `,"species":{"generation":"g"}}]}`,
			"want 1 or 2",
		},
		{"malformed json", `{`, "decode dataset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.json))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestEmbeddedConcepts(t *testing.T) {
	ds, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Concepts) == 0 {
		t.Fatal("no concepts loaded")
	}
	keys := map[string]bool{}
	for _, c := range ds.Concepts {
		keys[c.Key] = true
	}
	for _, want := range []string{"sleep", "paralysis", "poison", "rain"} {
		if !keys[want] {
			t.Errorf("concept %q missing", want)
		}
	}
}

func TestParseConceptsValidation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{"valid", `{"concepts":[{"key":"sleep","label":"Sleep","aliases":["asleep"],"terms":["sleep","puts the target to sleep"]}]}`, ""},
		{"bad key", `{"concepts":[{"key":"Sleep","label":"S","aliases":["a"],"terms":["t"]}]}`, "kebab-case"},
		{"duplicate concept", `{"concepts":[{"key":"a","label":"A","aliases":["x"],"terms":["t"]},{"key":"a","label":"A","aliases":["y"],"terms":["t"]}]}`, "duplicate concept"},
		{"empty label", `{"concepts":[{"key":"a","label":" ","aliases":["x"],"terms":["t"]}]}`, "empty label"},
		{"no aliases", `{"concepts":[{"key":"a","label":"A","aliases":[],"terms":["t"]}]}`, "no aliases"},
		{"no terms", `{"concepts":[{"key":"a","label":"A","aliases":["x"]}]}`, "no terms"},
		{"uppercase alias", `{"concepts":[{"key":"a","label":"A","aliases":["Asleep"],"terms":["t"]}]}`, "lowercase"},
		{"multi-word alias", `{"concepts":[{"key":"a","label":"A","aliases":["fall asleep"],"terms":["t"]}]}`, "single word"},
		{"duplicate alias", `{"concepts":[{"key":"a","label":"A","aliases":["x","x"],"terms":["t"]}]}`, "duplicate alias"},
		{"untrimmed term", `{"concepts":[{"key":"a","label":"A","aliases":["x"],"terms":[" t"]}]}`, "untrimmed"},
		{"unknown field", `{"concepts":[{"key":"a","label":"A","aliases":["x"],"terms":["t"],"typo":1}]}`, "decode concepts"},
		{"malformed", `{`, "decode concepts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConcepts([]byte(tt.json))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
