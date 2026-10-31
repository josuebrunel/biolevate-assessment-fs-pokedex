package testdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

// Recorder wraps a Bob executor and records every query issued through it, so
// tests can assert query counts (no N+1) and check query plans.
//
// Any test that runs its queries through a Recorder gets CheckPlans for free:
// add the call at the end of the test and new queries are covered automatically.
type Recorder struct {
	bob.Executor
	count   atomic.Int64
	mu      sync.Mutex
	queries map[string][]any // distinct SQL text -> first args seen
}

// NewRecorder wraps conn.
func NewRecorder(conn *sql.DB) *Recorder {
	return &Recorder{Executor: bob.NewDB(conn), queries: map[string][]any{}}
}

// QueryContext records the query, then runs it.
func (r *Recorder) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	r.count.Add(1)
	r.mu.Lock()
	if _, ok := r.queries[query]; !ok {
		r.queries[query] = args
	}
	r.mu.Unlock()
	return r.Executor.QueryContext(ctx, query, args...)
}

// Count is the number of queries issued so far.
func (r *Recorder) Count() int64 { return r.count.Load() }

// CheckPlans fails the test for every recorded query whose plan needs a
// sequential scan; see PlanProblems.
func (r *Recorder) CheckPlans(t *testing.T, conn *sql.DB) {
	t.Helper()
	problems, err := r.PlanProblems(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// PlanProblems runs EXPLAIN on every distinct recorded query with sequential
// scans disabled and describes each plan that still contains a Seq Scan. With
// seqscan off the planner only falls back to one when no index path exists, so
// this holds even on tiny tables where Postgres would rightly prefer a seq scan.
func (r *Recorder) PlanProblems(ctx context.Context, conn *sql.DB) ([]string, error) {
	session, err := conn.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()
	if _, err := session.ExecContext(ctx, "SET enable_seqscan = off"); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queries) == 0 {
		return nil, fmt.Errorf("no queries recorded")
	}
	var problems []string
	for query, args := range r.queries {
		var raw []byte
		if err := session.QueryRowContext(ctx, "EXPLAIN (FORMAT JSON) "+query, args...).Scan(&raw); err != nil {
			return nil, fmt.Errorf("explain %q: %w", query, err)
		}
		var plans []struct {
			Plan map[string]any `json:"Plan"`
		}
		if err := json.Unmarshal(raw, &plans); err != nil {
			return nil, err
		}
		for _, p := range plans {
			for _, rel := range seqScans(p.Plan) {
				problems = append(problems, fmt.Sprintf("sequential scan on %s, no usable index:\n  %s", rel, query))
			}
		}
	}
	return problems, nil
}

// seqScans returns the relations read by Seq Scan nodes anywhere in the plan.
func seqScans(node map[string]any) []string {
	var out []string
	if node["Node Type"] == "Seq Scan" {
		out = append(out, fmt.Sprint(node["Relation Name"]))
	}
	children, _ := node["Plans"].([]any)
	for _, c := range children {
		if m, ok := c.(map[string]any); ok {
			out = append(out, seqScans(m)...)
		}
	}
	return out
}
