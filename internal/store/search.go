package store

import (
	"context"
	"strings"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// containsMinLen is the shortest fragment matched anywhere inside a name. The
// trigram index serves LIKE '%x%' from three characters; shorter fragments
// would match nearly everything.
const containsMinLen = 3

// containsScore ranks a name that merely contains the fragment: above a pure
// typo match, below any prefix match (which scores 1 and more).
const containsScore = 0.5

// NameSearch finds entities by name with prefix, substring and typo tolerance.
// Prefix matches always outrank substring matches ("chu" finds "pikachu"),
// which outrank fuzzy trigram matches. Spaces in q are read as hyphens, so
// "mr m" finds "mr-mime".
func (s *Store) NameSearch(ctx context.Context, kind Kind, q string, limit int) ([]Hit, error) {
	if err := kind.valid(); err != nil {
		return nil, err
	}
	q = strings.Join(strings.Fields(strings.ToLower(q)), "-")
	if q == "" {
		return nil, nil
	}
	escaped := likeEscaper.Replace(q)
	prefix := escaped + "%"

	score := psql.Raw("similarity(name, ?) + CASE WHEN name LIKE ? THEN 1 ELSE 0 END AS score", q, prefix)
	match := psql.Raw("name LIKE ? OR name % ?", prefix, q)
	if len(q) >= containsMinLen {
		contains := "%" + escaped + "%"
		score = psql.Raw("similarity(name, ?) + CASE WHEN name LIKE ? THEN 1 WHEN name LIKE ? THEN ? ELSE 0 END AS score", q, prefix, contains, containsScore)
		match = psql.Raw("name LIKE ? OR name LIKE ? OR name % ?", prefix, contains, q)
	}

	query := psql.Select(
		sm.Columns("id", "name", score),
		sm.From(string(kind)),
		sm.Where(match),
		sm.OrderBy("score").Desc(), sm.OrderBy("name"),
		sm.Limit(limitOrDefault(limit)),
	)
	return s.hits(ctx, string(kind)+" names", query)
}

// Scoring weights for TextSearch.
const (
	// phraseWeight scales a phrase match: "puts the target to sleep" says far
	// more than the stem "sleep", which also occurs in "sleeps for two turns".
	phraseWeight = 3.0
	// statusMoveBoost favours status moves in move searches. Their text
	// describes the move's main effect, while a damaging move's text describes
	// a side effect ("has a chance to paralyze"). Cover-density ranks move in
	// steps of 0.1 to 0.2, so this separates ties without overriding relevance.
	statusMoveBoost = 0.15
)

// TextSearch runs full-text search over the weighted search_doc column. A row
// matches when it satisfies any of the terms. A single-word term matches that
// word's stem; a term with spaces is a phrase whose words must appear in order.
//
// The score is the sum of ts_rank_cd per term, with phrases weighted by
// phraseWeight, so a row matching a phrase and its words outranks one that only
// mentions a word. (Ranking the OR of all terms would ignore the phrase.)
// Move searches also apply statusMoveBoost.
func (s *Store) TextSearch(ctx context.Context, kind Kind, terms []string, limit int) ([]Hit, error) {
	if err := kind.valid(); err != nil {
		return nil, err
	}
	var queries, ranks []string
	var queryArgs, rankArgs []any
	for _, t := range terms {
		if t = strings.TrimSpace(t); t == "" {
			continue
		}
		fn, weight := "plainto_tsquery", 1.0
		if strings.ContainsAny(t, " \t") {
			fn, weight = "phraseto_tsquery", phraseWeight
		}
		tsquery := fn + "('english', ?)"
		queries = append(queries, tsquery)
		queryArgs = append(queryArgs, t)
		ranks = append(ranks, "?::float8 * ts_rank_cd(search_doc, "+tsquery+")")
		rankArgs = append(rankArgs, weight, t)
	}
	if len(queries) == 0 {
		return nil, nil
	}

	score := strings.Join(ranks, " + ")
	if kind == KindMove {
		// Cast explicitly: with an integer ELSE branch Postgres would infer the
		// parameter as an integer and silently truncate the boost to zero.
		score += " + CASE WHEN damage_class = 'status' THEN ?::float8 ELSE 0 END"
		rankArgs = append(rankArgs, statusMoveBoost)
	}

	query := psql.Select(
		sm.Columns("id", "name", psql.Raw("("+score+") AS score", rankArgs...)),
		sm.From(string(kind)),
		sm.Where(psql.Raw("search_doc @@ ("+strings.Join(queries, " || ")+")", queryArgs...)),
		sm.OrderBy("score").Desc(), sm.OrderBy("name"),
		sm.Limit(limitOrDefault(limit)),
	)
	return s.hits(ctx, string(kind)+" text", query)
}
