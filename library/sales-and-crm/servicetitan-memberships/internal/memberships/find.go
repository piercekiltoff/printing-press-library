package memberships

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// FindResult is one ranked membership-finder hit, shaped for an ops user
// looking up a customer's membership by a fuzzy description.
type FindResult struct {
	ID               int64   `json:"id"`
	CustomerID       int64   `json:"customer_id"`
	MembershipTypeID int64   `json:"membership_type_id"`
	MembershipType   string  `json:"membership_type"`
	Status           string  `json:"status"`
	FollowUpStatus   string  `json:"follow_up_status"`
	Active           bool    `json:"active"`
	ImportID         string  `json:"import_id"`
	Memo             string  `json:"memo"`
	Score            float64 `json:"score"`
	MatchedOn        string  `json:"matched_on"`
	MatchedValue     string  `json:"matched_value"`
}

// Find runs a forgiving ranked search across memberships for a natural-
// language query. Each membership is scored on customerId, importId, memo,
// any customFields[].typeName/value, and the joined membership-type name;
// the best field wins. Only memberships scoring at or above minScore are
// returned, so a nonsense query yields an empty result rather than weak
// junk; pass minScore <= 0 to keep every positive hit. Results are sorted
// by score descending and capped at limit (default 15).
func Find(db *store.Store, query string, minScore float64, limit int) ([]FindResult, error) {
	if limit <= 0 {
		limit = 15
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	memberships, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	types, err := LoadMembershipTypes(db)
	if err != nil {
		return nil, err
	}

	var out []FindResult
	for _, m := range memberships {
		best, matchedField, matchedValue := 0.0, "", ""
		consider := func(field, value string) {
			if value == "" {
				return
			}
			s := similarity(q, value)
			if cov := tokenCoverage(q, value); cov > s {
				s = cov
			}
			if nq, nv := normalize(q), normalize(value); nq != "" && strings.Contains(nv, nq) && s < 0.85 {
				s = 0.85
			}
			if s > best {
				best, matchedField, matchedValue = s, field, value
			}
		}
		consider("customer-id", fmt.Sprintf("%d", m.CustomerID))
		consider("import-id", m.ImportID)
		consider("memo", m.Memo)
		for _, cf := range m.CustomFields {
			consider("custom-field:"+cf.TypeName, cf.Value)
			consider("custom-field-name", cf.TypeName)
		}
		mtName := ""
		if mt, ok := types[m.MembershipTypeID]; ok {
			mtName = mt.Name
			if mt.DisplayName != "" {
				mtName = mt.DisplayName
			}
			consider("membership-type", mtName)
		}
		if best <= 0 || best < minScore {
			continue
		}
		out = append(out, FindResult{
			ID: m.ID, CustomerID: m.CustomerID, MembershipTypeID: m.MembershipTypeID,
			MembershipType: mtName, Status: m.Status, FollowUpStatus: m.FollowUpStatus,
			Active: m.Active, ImportID: m.ImportID, Memo: m.Memo,
			Score: round2(best), MatchedOn: matchedField, MatchedValue: matchedValue,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].Active != out[j].Active {
			return out[i].Active
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// normalize lower-cases s and reduces every run of non-alphanumeric to a
// single space, then trims. Shared canonical form for fuzzy matching.
func normalize(s string) string {
	var b strings.Builder
	lastSpace := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
		} else if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func tokens(s string) []string {
	fields := strings.Fields(normalize(s))
	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

// tokenCoverage is the asymmetric overlap of query against target in [0,1]:
// the fraction of query tokens that also appear in target. Good for
// "short description against longer title" — the natural-language find case.
func tokenCoverage(query, target string) float64 {
	tq, tt := tokens(query), tokens(target)
	if len(tq) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(tt))
	for _, t := range tt {
		set[t] = struct{}{}
	}
	hit := 0
	for _, t := range tq {
		if _, ok := set[t]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(tq))
}

func jaccard(a, b string) float64 {
	ta, tb := tokens(a), tokens(b)
	if len(ta) == 0 && len(tb) == 0 {
		return 1
	}
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(ta))
	for _, t := range ta {
		set[t] = struct{}{}
	}
	inter := 0
	for _, t := range tb {
		if _, ok := set[t]; ok {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func levenshteinRatio(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	maxLen := len([]rune(a))
	if l := len([]rune(b)); l > maxLen {
		maxLen = l
	}
	if maxLen == 0 {
		return 1
	}
	return 1 - float64(levenshtein(a, b))/float64(maxLen)
}

// similarity is the general-purpose fuzzy score in [0,1].
func similarity(a, b string) float64 {
	na, nb := normalize(a), normalize(b)
	if na == "" && nb == "" {
		return 1
	}
	if na == "" || nb == "" {
		return 0
	}
	score := jaccard(a, b)
	if lr := levenshteinRatio(na, nb); lr > score {
		score = lr
	}
	if na != nb && (strings.Contains(na, nb) || strings.Contains(nb, na)) {
		if 0.9 > score {
			score = 0.9
		}
	}
	return score
}
