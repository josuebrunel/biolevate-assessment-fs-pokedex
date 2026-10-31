// Package golden scores a search engine against a fixed set of queries with
// known good answers. The cases are data (cases.json), the runner works with
// any Searcher, and the report gives recall, mean reciprocal rank and a pass
// rate per category, so a change to ranking, or a new retriever such as a
// vector one, is judged on the same yardstick.
package golden

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/search"
	"github.com/josuebrunel/biolevate-assessment-fs-pokedex/internal/store"
)

//go:embed cases.json
var casesJSON []byte

// Searcher is what the suite scores: the engine, or any stand-in with the
// same shape.
type Searcher interface {
	Search(ctx context.Context, input string, opts search.Options) (search.Response, error)
}

// Expect names a result that must (or, in Forbid, must not) appear among the
// first Within results of its kind. Within 0 means anywhere for Forbid and is
// invalid for Expect.
type Expect struct {
	Kind   store.Kind `json:"kind"`
	Name   string     `json:"name"`
	Within int        `json:"within"`
}

// Case is one query with its expected answers.
type Case struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Query       string   `json:"query"`
	Expect      []Expect `json:"expect,omitempty"`
	Forbid      []Expect `json:"forbid,omitempty"`
	ExpectEmpty bool     `json:"expect_empty,omitempty"` // the engine must return no results
	// Known documents a miss the engine has today. A known case must fail, so
	// that fixing it forces the flag, and the limit it records, to be removed.
	Known string `json:"known,omitempty"`
}

// LoadCases returns the embedded cases after validating them.
func LoadCases() ([]Case, error) {
	var cases []Case
	if err := json.Unmarshal(casesJSON, &cases); err != nil {
		return nil, fmt.Errorf("parse golden cases: %w", err)
	}
	seen := map[string]bool{}
	for _, c := range cases {
		switch {
		case c.Name == "" || c.Category == "":
			return nil, fmt.Errorf("case %q: name and category are required", c.Name)
		case seen[c.Name]:
			return nil, fmt.Errorf("case %q is duplicated", c.Name)
		case len(c.Expect) == 0 && !c.ExpectEmpty && len(c.Forbid) == 0:
			return nil, fmt.Errorf("case %q expects nothing", c.Name)
		case c.ExpectEmpty && len(c.Expect) > 0:
			return nil, fmt.Errorf("case %q cannot expect both results and none", c.Name)
		}
		seen[c.Name] = true
		for _, e := range c.Expect {
			if e.Within <= 0 || e.Name == "" || e.Kind == "" {
				return nil, fmt.Errorf("case %q: expectation %+v needs kind, name and within > 0", c.Name, e)
			}
		}
	}
	return cases, nil
}

// Outcome is the result of one case.
type Outcome struct {
	Case     Case
	Pass     bool
	Problems []string
	Hits     int     // expectations met
	RR       float64 // sum of reciprocal ranks over the expectations
}

// Report is the outcome of a whole run.
type Report struct {
	Outcomes []Outcome
}

// Run executes every case and returns the report. A search error fails the
// case rather than the run, so one bad query cannot hide the others.
func Run(ctx context.Context, s Searcher, cases []Case) Report {
	var r Report
	for _, c := range cases {
		r.Outcomes = append(r.Outcomes, runCase(ctx, s, c))
	}
	return r
}

func runCase(ctx context.Context, s Searcher, c Case) Outcome {
	o := Outcome{Case: c}
	resp, err := s.Search(ctx, c.Query, search.Options{})
	if err != nil {
		o.Problems = []string{"search failed: " + err.Error()}
		return o
	}

	names := namesByKind(resp)
	empty := len(resp.Groups) == 0
	if c.ExpectEmpty && !empty {
		o.Problems = append(o.Problems, "expected no results")
	}
	for _, e := range c.Expect {
		rank := slices.Index(names[e.Kind], e.Name) + 1 // 0 when absent
		switch {
		case rank == 0:
			o.Problems = append(o.Problems, fmt.Sprintf("%s %q missing", e.Kind, e.Name))
		case rank > e.Within:
			o.Problems = append(o.Problems, fmt.Sprintf("%s %q at rank %d, want within %d", e.Kind, e.Name, rank, e.Within))
		default:
			o.Hits++
		}
		if rank > 0 {
			o.RR += 1 / float64(rank)
		}
	}
	for _, f := range c.Forbid {
		limit := f.Within
		if limit <= 0 {
			limit = len(names[f.Kind])
		}
		if rank := slices.Index(names[f.Kind], f.Name) + 1; rank > 0 && rank <= limit {
			o.Problems = append(o.Problems, fmt.Sprintf("%s %q must not appear, found at rank %d", f.Kind, f.Name, rank))
		}
	}
	o.Pass = len(o.Problems) == 0
	return o
}

// namesByKind flattens the groups into one ranked name list per kind.
func namesByKind(r search.Response) map[store.Kind][]string {
	out := map[store.Kind][]string{}
	for _, g := range r.Groups {
		for _, it := range g.Items {
			out[g.Kind] = append(out[g.Kind], it.Name)
		}
	}
	return out
}

// Stats summarises a set of outcomes.
type Stats struct {
	Cases        int
	Passed       int
	Expectations int
	Hits         int
	RR           float64
}

// PassRate is the share of cases that passed.
func (s Stats) PassRate() float64 { return ratio(float64(s.Passed), float64(s.Cases)) }

// Recall is the share of expectations met within their limit.
func (s Stats) Recall() float64 { return ratio(float64(s.Hits), float64(s.Expectations)) }

// MRR is the mean reciprocal rank of the expected results (0 when absent).
func (s Stats) MRR() float64 { return ratio(s.RR, float64(s.Expectations)) }

func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// Summary scores the outcomes that pass the filter.
func (r Report) Summary(keep func(Outcome) bool) Stats {
	var s Stats
	for _, o := range r.Outcomes {
		if keep != nil && !keep(o) {
			continue
		}
		s.Cases++
		if o.Pass {
			s.Passed++
		}
		s.Expectations += len(o.Case.Expect)
		s.Hits += o.Hits
		s.RR += o.RR
	}
	return s
}

// Core are the outcomes of cases without a documented limit: the ones the
// engine is expected to get right.
func Core(o Outcome) bool { return o.Case.Known == "" }

// Categories lists the categories in the report, sorted.
func (r Report) Categories() []string {
	set := map[string]bool{}
	for _, o := range r.Outcomes {
		set[o.Case.Category] = true
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Failures returns the problems worth acting on: core cases that fail, and
// known cases that now pass (so their flag is stale).
func (r Report) Failures() []string {
	var out []string
	for _, o := range r.Outcomes {
		switch {
		case o.Case.Known == "" && !o.Pass:
			out = append(out, fmt.Sprintf("%s (%q): %s", o.Case.Name, o.Case.Query, strings.Join(o.Problems, "; ")))
		case o.Case.Known != "" && o.Pass:
			out = append(out, fmt.Sprintf("%s (%q) is marked known (%s) but passes: remove the flag", o.Case.Name, o.Case.Query, o.Case.Known))
		}
	}
	return out
}

// String renders the report as a table: one row per category, the totals, and
// the known limits with their reasons.
func (r Report) String() string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "category\tcases\tpassed\trecall\tMRR")
	row := func(name string, s Stats) {
		if s.Expectations == 0 {
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t-\t-\n", name, s.Cases, s.Passed)
			return
		}
		_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%.2f\t%.2f\n", name, s.Cases, s.Passed, s.Recall(), s.MRR())
	}
	for _, c := range r.Categories() {
		row(c, r.Summary(func(o Outcome) bool { return o.Case.Category == c && Core(o) }))
	}
	row("core total", r.Summary(Core))
	row("all (incl. known limits)", r.Summary(nil))
	_ = w.Flush()

	var known []Outcome
	for _, o := range r.Outcomes {
		if !Core(o) {
			known = append(known, o)
		}
	}
	if len(known) > 0 {
		b.WriteString("\nknown limits:\n")
		for _, o := range known {
			_, _ = fmt.Fprintf(&b, "  %-28s %s\n", o.Case.Query, o.Case.Known)
		}
	}
	return b.String()
}
