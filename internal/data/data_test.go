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
