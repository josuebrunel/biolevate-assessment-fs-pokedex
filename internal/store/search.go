package store

import (
	"context"
	"strings"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// NameSearch finds entities by name with prefix and typo tolerance. Prefix
// matches always outrank fuzzy trigram matches. Spaces in q are read as
// hyphens, so "mr m" finds "mr-mime".
func (s *Store) NameSearch(ctx context.Context, kind Kind, q string, limit int) ([]Hit, error) {
	if err := kind.valid(); err != nil {
		return nil, err
	}
	q = strings.Join(strings.Fields(strings.ToLower(q)), "-")
	if q == "" {
		return nil, nil
	}
	prefix := likeEscaper.Replace(q) + "%"

	query := psql.Select(
		sm.Columns("id", "name", psql.Raw("similarity(name, ?) + CASE WHEN name LIKE ? THEN 1 ELSE 0 END AS score", q, prefix)),
		sm.From(string(kind)),
		sm.Where(psql.Raw("name LIKE ? OR name % ?", prefix, q)),
		sm.OrderBy("score").Desc(), sm.OrderBy("name"),
		sm.Limit(limitOrDefault(limit)),
	)
	return s.hits(ctx, string(kind)+" names", query)
}

// TextSearch runs full-text search over the weighted search_doc column. A row
// matches when it satisfies any of the terms. A single-word term matches that
// word's stem; a term with spaces is a phrase whose words must appear in order
// ("puts the target to sleep"), so phrases rank causal text above mere
// mentions. Hits are ordered by ts_rank_cd, which rewards matching several
// terms.
func (s *Store) TextSearch(ctx context.Context, kind Kind, terms []string, limit int) ([]Hit, error) {
	if err := kind.valid(); err != nil {
		return nil, err
	}
	var parts []string
	var args []any
	for _, t := range terms {
		if t = strings.TrimSpace(t); t != "" {
			fn := "plainto_tsquery"
			if strings.ContainsAny(t, " \t") {
				fn = "phraseto_tsquery"
			}
			parts = append(parts, fn+"('english', ?)")
			args = append(args, t)
		}
	}
	if len(parts) == 0 {
		return nil, nil
	}
	tsquery := "(" + strings.Join(parts, " || ") + ")"

	query := psql.Select(
		sm.Columns("id", "name", psql.Raw("ts_rank_cd(search_doc, "+tsquery+") AS score", args...)),
		sm.From(string(kind)),
		sm.Where(psql.Raw("search_doc @@ "+tsquery, args...)),
		sm.OrderBy("score").Desc(), sm.OrderBy("name"),
		sm.Limit(limitOrDefault(limit)),
	)
	return s.hits(ctx, string(kind)+" text", query)
}
