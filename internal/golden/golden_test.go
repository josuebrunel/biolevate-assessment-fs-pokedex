package golden_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/golden"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/testdb"
)

// TestGolden scores the real engine against the golden cases. Every core case
// must pass; a known limit must still fail, so fixing one forces its flag, and
// the limit it documents, to be removed. Run with -v (make eval) for the report.
func TestGolden(t *testing.T) {
	conn := testdb.Loaded(t)
	s := store.NewFromSQL(conn)
	concepts, err := search.LoadConceptMap(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	engine := search.NewEngine(search.NewLexical(s, search.WithExpander(concepts.Expand)), s, concepts)

	cases, err := golden.LoadCases()
	if err != nil {
		t.Fatal(err)
	}
	report := golden.Run(context.Background(), engine, cases)
	t.Logf("\n%s", report)

	for _, f := range report.Failures() {
		t.Error(f)
	}
}

// fake answers each query with a fixed response.
type fake map[string]search.Response

func (f fake) Search(_ context.Context, q string, _ search.Options) (search.Response, error) {
	if q == "boom" {
		return search.Response{}, errors.New("db down")
	}
	return f[q], nil
}

func group(kind store.Kind, names ...string) search.Group {
	g := search.Group{Title: string(kind), Kind: kind}
	for _, n := range names {
		g.Items = append(g.Items, search.Item{Kind: kind, Name: n})
	}
	return g
}

func TestRunnerScoring(t *testing.T) {
	s := fake{
		"hit":    {Groups: []search.Group{group(store.KindMove, "a", "b", "c")}},
		"late":   {Groups: []search.Group{group(store.KindMove, "a", "b", "c")}},
		"absent": {Groups: []search.Group{group(store.KindMove, "a")}},
		"empty":  {},
		"some":   {Groups: []search.Group{group(store.KindPokemon, "p")}},
		"forbid": {Groups: []search.Group{group(store.KindPokemon, "x", "y")}},
	}
	m := func(name string, within int) golden.Expect {
		return golden.Expect{Kind: store.KindMove, Name: name, Within: within}
	}
	cases := []golden.Case{
		{Name: "hit", Category: "c", Query: "hit", Expect: []golden.Expect{m("b", 2)}},
		{Name: "late", Category: "c", Query: "late", Expect: []golden.Expect{m("c", 2)}},
		{Name: "absent", Category: "c", Query: "absent", Expect: []golden.Expect{m("z", 5)}},
		{Name: "empty ok", Category: "c", Query: "empty", ExpectEmpty: true},
		{Name: "empty bad", Category: "c", Query: "some", ExpectEmpty: true},
		{Name: "forbid", Category: "c", Query: "forbid", Forbid: []golden.Expect{{Kind: store.KindPokemon, Name: "y", Within: 2}}},
		{Name: "error", Category: "c", Query: "boom", ExpectEmpty: true},
		{Name: "known and failing", Category: "c", Query: "absent", Expect: []golden.Expect{m("z", 5)}, Known: "documented"},
		{Name: "known but fixed", Category: "c", Query: "hit", Expect: []golden.Expect{m("a", 1)}, Known: "stale"},
	}
	r := golden.Run(context.Background(), s, cases)

	wantPass := []bool{true, false, false, true, false, false, false, false, true}
	for i, o := range r.Outcomes {
		if o.Pass != wantPass[i] {
			t.Errorf("%s: pass = %v, want %v (%v)", o.Case.Name, o.Pass, wantPass[i], o.Problems)
		}
	}
	// hit: rank 2 of 3 → 0.5; late: rank 3 → 1/3 (ranked but outside the limit).
	if got := r.Outcomes[0].RR; got != 0.5 {
		t.Errorf("RR = %v, want 0.5", got)
	}
	if got := r.Outcomes[1].RR; got < 0.33 || got > 0.34 {
		t.Errorf("late RR = %v, want 1/3", got)
	}

	failures := strings.Join(r.Failures(), "\n")
	for _, want := range []string{"late", "absent", "empty bad", "forbid", "error", "known but fixed"} {
		if !strings.Contains(failures, want) && !strings.Contains(failures, "stale") {
			t.Errorf("failures lack %q:\n%s", want, failures)
		}
	}
	if strings.Contains(failures, "known and failing") {
		t.Error("a documented known limit must not be reported as a failure")
	}
	if core := r.Summary(golden.Core); core.Cases != 7 || core.Passed != 2 {
		t.Errorf("core summary = %+v", core)
	}
	if out := r.String(); !strings.Contains(out, "core total") || !strings.Contains(out, "documented") {
		t.Errorf("report lacks totals or known limits:\n%s", out)
	}
}

func TestLoadCasesValidates(t *testing.T) {
	cases, err := golden.LoadCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 40 {
		t.Errorf("only %d cases; the suite should keep covering the four challenges", len(cases))
	}
	for _, want := range []string{"names and typos", "criteria and comparison", "vocabulary-free discovery", "strategy bundles", "negatives and hostile input"} {
		found := false
		for _, c := range cases {
			found = found || c.Category == want
		}
		if !found {
			t.Errorf("no case in category %q", want)
		}
	}
}
