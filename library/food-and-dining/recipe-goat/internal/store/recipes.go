// Recipe-aggregation persistence layer (Phase 3).
//
// A note on typing: rather than importing internal/recipes (which would create
// an import cycle — the recipes package doesn't depend on store, but callers
// like internal/cli import both), this file defines a `StoredRecipe` DTO that
// mirrors the wire shape. CLI commands translate between recipes.Recipe and
// StoredRecipe using the conversion helpers at the top.

package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// StoredRecipe is the shape the store reads/writes. It is intentionally a
// plain struct (no time.Time in fetched_at column — we use unix seconds) to
// keep SQL scanning simple.
type StoredRecipe struct {
	ID               int64
	URL              string
	Site             string
	Title            string
	Author           string
	ImageURL         string
	TotalTimeS       int
	PrepTimeS        int
	CookTimeS        int
	Servings         int
	Rating           float64
	ReviewCount      int
	NutritionJSON    string // raw JSON of map[string]string
	IngredientsJSON  string // raw JSON of []string
	InstructionsJSON string
	Description      string
	KeywordsJSON     string
	CategoriesJSON   string
	CuisinesJSON     string
	FetchedAt        time.Time

	// Derived/convenience.
	Ingredients []string `json:"-"`
}

// CookLogEntry records one cooking session.
type CookLogEntry struct {
	ID       int64     `json:"id"`
	RecipeID int64     `json:"recipeId"`
	CookedAt time.Time `json:"cookedAt"`
	Rating   int       `json:"rating,omitempty"`
	Notes    string    `json:"notes,omitempty"`
}

// MealPlanEntry is one slot in the weekly plan.
type MealPlanEntry struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"` // YYYY-MM-DD
	Meal     string `json:"meal"` // breakfast|lunch|dinner|snack
	RecipeID int64  `json:"recipeId"`
	Title    string `json:"title,omitempty"`
}

// RecipeMatch is a pantry-match result (N have, M missing).
type RecipeMatch struct {
	Recipe  *StoredRecipe `json:"recipe"`
	Missing []string      `json:"missing"`
}

// SaveRecipe upserts a recipe by URL and returns its row ID.
func (s *Store) SaveRecipe(r *StoredRecipe) (int64, error) {
	if r.URL == "" {
		return 0, fmt.Errorf("recipe URL is required")
	}
	if r.FetchedAt.IsZero() {
		r.FetchedAt = time.Now().UTC()
	}
	if r.IngredientsJSON == "" && len(r.Ingredients) > 0 {
		b, _ := json.Marshal(r.Ingredients)
		r.IngredientsJSON = string(b)
	}
	if r.IngredientsJSON == "" {
		r.IngredientsJSON = "[]"
	}
	if r.InstructionsJSON == "" {
		r.InstructionsJSON = "[]"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Upsert pattern: try UPDATE first, then INSERT if no row.
	var id int64
	row := tx.QueryRow(`SELECT id FROM recipes WHERE url = ?`, r.URL)
	err = row.Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if err == sql.ErrNoRows {
		res, err := tx.Exec(`INSERT INTO recipes
			(url, site, title, author, image_url, total_time_s, prep_time_s, cook_time_s, servings, rating, review_count,
			 nutrition_json, ingredients_json, instructions_json, description, keywords_json, categories_json, cuisines_json, fetched_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.URL, r.Site, r.Title, nullString(r.Author), nullString(r.ImageURL),
			r.TotalTimeS, r.PrepTimeS, r.CookTimeS, r.Servings, r.Rating, r.ReviewCount,
			nullString(r.NutritionJSON), r.IngredientsJSON, r.InstructionsJSON,
			nullString(r.Description), nullString(r.KeywordsJSON), nullString(r.CategoriesJSON), nullString(r.CuisinesJSON),
			r.FetchedAt.Unix())
		if err != nil {
			return 0, fmt.Errorf("insert recipe: %w", err)
		}
		id, _ = res.LastInsertId()
	} else {
		// FTS content='recipes' table uses AFTER INSERT/DELETE triggers — to
		// keep FTS in sync on UPDATE, emit delete+insert of the FTS external
		// content manually around the UPDATE.
		_, err := tx.Exec(
			`INSERT INTO recipes_fts(recipes_fts, rowid, title, author, ingredients, instructions, description, keywords)
			 SELECT 'delete', id, title, author, ingredients_json, instructions_json, description, keywords_json FROM recipes WHERE id = ?`, id)
		if err != nil {
			// FTS maintenance is non-fatal.
			_ = err
		}
		_, err = tx.Exec(`UPDATE recipes SET
			site=?, title=?, author=?, image_url=?, total_time_s=?, prep_time_s=?, cook_time_s=?, servings=?, rating=?, review_count=?,
			nutrition_json=?, ingredients_json=?, instructions_json=?, description=?, keywords_json=?, categories_json=?, cuisines_json=?, fetched_at=?
			WHERE id=?`,
			r.Site, r.Title, nullString(r.Author), nullString(r.ImageURL),
			r.TotalTimeS, r.PrepTimeS, r.CookTimeS, r.Servings, r.Rating, r.ReviewCount,
			nullString(r.NutritionJSON), r.IngredientsJSON, r.InstructionsJSON,
			nullString(r.Description), nullString(r.KeywordsJSON), nullString(r.CategoriesJSON), nullString(r.CuisinesJSON),
			r.FetchedAt.Unix(), id)
		if err != nil {
			return 0, fmt.Errorf("update recipe: %w", err)
		}
		_, _ = tx.Exec(
			`INSERT INTO recipes_fts(rowid, title, author, ingredients, instructions, description, keywords)
			 SELECT id, title, author, ingredients_json, instructions_json, description, keywords_json FROM recipes WHERE id = ?`, id)
	}
	return id, tx.Commit()
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// scanRecipe reads one row using the canonical column order.
func scanRecipe(scanner interface {
	Scan(dest ...any) error
}) (*StoredRecipe, error) {
	var r StoredRecipe
	var author, imageURL, description, nutrition, keywords, categories, cuisines sql.NullString
	var fetchedUnix int64
	err := scanner.Scan(
		&r.ID, &r.URL, &r.Site, &r.Title, &author, &imageURL,
		&r.TotalTimeS, &r.PrepTimeS, &r.CookTimeS, &r.Servings, &r.Rating, &r.ReviewCount,
		&nutrition, &r.IngredientsJSON, &r.InstructionsJSON,
		&description, &keywords, &categories, &cuisines, &fetchedUnix,
	)
	if err != nil {
		return nil, err
	}
	r.Author = author.String
	r.ImageURL = imageURL.String
	r.Description = description.String
	r.NutritionJSON = nutrition.String
	r.KeywordsJSON = keywords.String
	r.CategoriesJSON = categories.String
	r.CuisinesJSON = cuisines.String
	r.FetchedAt = time.Unix(fetchedUnix, 0).UTC()
	if r.IngredientsJSON != "" {
		_ = json.Unmarshal([]byte(r.IngredientsJSON), &r.Ingredients)
	}
	return &r, nil
}

const recipeCols = `id, url, site, title, author, image_url, total_time_s, prep_time_s, cook_time_s, servings, rating, review_count,
	nutrition_json, ingredients_json, instructions_json, description, keywords_json, categories_json, cuisines_json, fetched_at`

// GetRecipeByID returns the recipe with the given row ID, or (nil, nil) when missing.
func (s *Store) GetRecipeByID(id int64) (*StoredRecipe, error) {
	row := s.db.QueryRow(`SELECT `+recipeCols+` FROM recipes WHERE id = ?`, id)
	r, err := scanRecipe(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetRecipeByURL returns the recipe with the given URL, or (nil, nil) when missing.
func (s *Store) GetRecipeByURL(u string) (*StoredRecipe, error) {
	row := s.db.QueryRow(`SELECT `+recipeCols+` FROM recipes WHERE url = ?`, u)
	r, err := scanRecipe(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// ListRecipes lists recipes with optional filters. Empty strings = no filter.
func (s *Store) ListRecipes(tag, site, author string, limit, offset int) ([]*StoredRecipe, error) {
	if limit <= 0 {
		limit = 100
	}
	var where []string
	var args []any
	q := `SELECT ` + recipeCols + ` FROM recipes `
	if tag != "" {
		q += ` JOIN cookbook_tags t ON t.recipe_id = recipes.id`
		where = append(where, `t.tag = ?`)
		args = append(args, tag)
	}
	if site != "" {
		where = append(where, `recipes.site = ?`)
		args = append(args, site)
	}
	if author != "" {
		where = append(where, `LOWER(recipes.author) LIKE ?`)
		args = append(args, "%"+strings.ToLower(author)+"%")
	}
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY recipes.fetched_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*StoredRecipe{}
	for rows.Next() {
		r, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SearchRecipesFTS runs an FTS5 MATCH query and returns ranked results.
func (s *Store) SearchRecipesFTS(query string, limit int) ([]*StoredRecipe, error) {
	if limit <= 0 {
		limit = 20
	}
	// Escape double quotes in the query for FTS5.
	ftsQuery := strings.ReplaceAll(query, `"`, `""`)
	rows, err := s.db.Query(
		`SELECT `+recipeCols+` FROM recipes
		 WHERE id IN (SELECT rowid FROM recipes_fts WHERE recipes_fts MATCH ? ORDER BY rank LIMIT ?)`,
		ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*StoredRecipe{}
	for rows.Next() {
		r, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RemoveRecipe deletes a recipe (cascades to tags, cook_log, meal_plan).
func (s *Store) RemoveRecipe(id int64) error {
	_, err := s.db.Exec(`DELETE FROM recipes WHERE id = ?`, id)
	return err
}

// TagRecipe attaches the given tags (idempotent — existing pairs are ignored).
func (s *Store) TagRecipe(id int64, tags []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO cookbook_tags(recipe_id, tag) VALUES (?, ?)`, id, t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UntagRecipe removes one tag from the recipe.
func (s *Store) UntagRecipe(id int64, tag string) error {
	_, err := s.db.Exec(`DELETE FROM cookbook_tags WHERE recipe_id=? AND tag=?`, id, strings.TrimSpace(strings.ToLower(tag)))
	return err
}

// GetTags returns all tags for a recipe, sorted alphabetically.
func (s *Store) GetTags(id int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT tag FROM cookbook_tags WHERE recipe_id = ? ORDER BY tag`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// LogCook records a cooking session (rating 0 and empty notes are allowed).
// Verifies recipeID exists first — SQLite FK enforcement via driver DSN is
// unreliable across modernc.org/sqlite versions, so we validate explicitly.
func (s *Store) LogCook(recipeID int64, rating int, notes string, cookedAt time.Time) error {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM recipes WHERE id = ?`, recipeID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("recipe id %d not found", recipeID)
	}
	if err != nil {
		return err
	}
	if cookedAt.IsZero() {
		cookedAt = time.Now().UTC()
	}
	_, err = s.db.Exec(`INSERT INTO cook_log(recipe_id, cooked_at, rating, notes) VALUES (?, ?, ?, ?)`,
		recipeID, cookedAt.Unix(), rating, nullString(notes))
	return err
}

// CookLogFor returns all cook log entries for a recipe, newest first.
func (s *Store) CookLogFor(recipeID int64) ([]CookLogEntry, error) {
	rows, err := s.db.Query(`SELECT id, recipe_id, cooked_at, rating, notes FROM cook_log WHERE recipe_id = ? ORDER BY cooked_at DESC`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CookLogEntry{}
	for rows.Next() {
		var e CookLogEntry
		var ts int64
		var rating sql.NullInt64
		var notes sql.NullString
		if err := rows.Scan(&e.ID, &e.RecipeID, &ts, &rating, &notes); err != nil {
			return nil, err
		}
		e.CookedAt = time.Unix(ts, 0).UTC()
		e.Rating = int(rating.Int64)
		e.Notes = notes.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// CookLogAll returns recent cook log entries across all recipes.
func (s *Store) CookLogAll(limit int, since time.Time) ([]CookLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows *sql.Rows
	var err error
	if since.IsZero() {
		rows, err = s.db.Query(`SELECT id, recipe_id, cooked_at, rating, notes FROM cook_log ORDER BY cooked_at DESC LIMIT ?`, limit)
	} else {
		rows, err = s.db.Query(`SELECT id, recipe_id, cooked_at, rating, notes FROM cook_log WHERE cooked_at >= ? ORDER BY cooked_at DESC LIMIT ?`, since.Unix(), limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CookLogEntry{}
	for rows.Next() {
		var e CookLogEntry
		var ts int64
		var rating sql.NullInt64
		var notes sql.NullString
		if err := rows.Scan(&e.ID, &e.RecipeID, &ts, &rating, &notes); err != nil {
			return nil, err
		}
		e.CookedAt = time.Unix(ts, 0).UTC()
		e.Rating = int(rating.Int64)
		e.Notes = notes.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// LastCookedMap returns the most recent cook time for each recipe ID.
// IDs absent from the log are omitted from the result map.
func (s *Store) LastCookedMap(recipeIDs []int64) (map[int64]time.Time, error) {
	out := map[int64]time.Time{}
	if len(recipeIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(recipeIDs))
	args := make([]any, len(recipeIDs))
	for i, id := range recipeIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT recipe_id, MAX(cooked_at) FROM cook_log WHERE recipe_id IN (` + strings.Join(placeholders, ",") + `) GROUP BY recipe_id`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, ts int64
		if err := rows.Scan(&id, &ts); err != nil {
			return nil, err
		}
		out[id] = time.Unix(ts, 0).UTC()
	}
	return out, rows.Err()
}

// SetMealPlan upserts a plan slot.
func (s *Store) SetMealPlan(date, meal string, recipeID int64) error {
	_, err := s.db.Exec(`INSERT INTO meal_plan(date, meal, recipe_id) VALUES (?, ?, ?)
		ON CONFLICT(date, meal) DO UPDATE SET recipe_id = excluded.recipe_id`, date, meal, recipeID)
	return err
}

// RemoveMealPlan clears the (date, meal) slot.
func (s *Store) RemoveMealPlan(date, meal string) error {
	_, err := s.db.Exec(`DELETE FROM meal_plan WHERE date = ? AND meal = ?`, date, meal)
	return err
}

// GetMealPlan returns plan entries in [from, to] (inclusive, YYYY-MM-DD).
func (s *Store) GetMealPlan(from, to string) ([]MealPlanEntry, error) {
	rows, err := s.db.Query(`SELECT m.id, m.date, m.meal, m.recipe_id, COALESCE(r.title,'')
		FROM meal_plan m LEFT JOIN recipes r ON r.id = m.recipe_id
		WHERE m.date BETWEEN ? AND ? ORDER BY m.date, m.meal`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MealPlanEntry{}
	for rows.Next() {
		var e MealPlanEntry
		if err := rows.Scan(&e.ID, &e.Date, &e.Meal, &e.RecipeID, &e.Title); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// KidExcluded returns the kid-exclusion list.
func (s *Store) KidExcluded() ([]string, error) {
	rows, err := s.db.Query(`SELECT ingredient FROM kid_excluded_ingredients ORDER BY ingredient`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var i string
		if err := rows.Scan(&i); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// KidExcludedAdd adds an ingredient (idempotent).
func (s *Store) KidExcludedAdd(ing string) error {
	ing = strings.TrimSpace(strings.ToLower(ing))
	if ing == "" {
		return fmt.Errorf("empty ingredient")
	}
	_, err := s.db.Exec(`INSERT OR IGNORE INTO kid_excluded_ingredients(ingredient) VALUES (?)`, ing)
	return err
}

// KidExcludedRemove removes an ingredient.
func (s *Store) KidExcludedRemove(ing string) error {
	_, err := s.db.Exec(`DELETE FROM kid_excluded_ingredients WHERE ingredient = ?`, strings.TrimSpace(strings.ToLower(ing)))
	return err
}

// MatchByIngredients returns recipes whose ingredient list can be assembled
// from `have` with at most `missingMax` missing items. Matching is
// case-insensitive substring per ingredient line.
func (s *Store) MatchByIngredients(have []string, missingMax int) ([]RecipeMatch, error) {
	if missingMax < 0 {
		missingMax = 0
	}
	haveLower := make([]string, 0, len(have))
	for _, h := range have {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			haveLower = append(haveLower, h)
		}
	}
	recipes, err := s.ListRecipes("", "", "", 10000, 0)
	if err != nil {
		return nil, err
	}
	out := []RecipeMatch{}
	for _, r := range recipes {
		if len(r.Ingredients) == 0 {
			continue
		}
		missing := []string{}
		for _, line := range r.Ingredients {
			lower := strings.ToLower(line)
			matched := false
			for _, h := range haveLower {
				if strings.Contains(lower, h) {
					matched = true
					break
				}
			}
			if !matched {
				missing = append(missing, line)
			}
		}
		if len(missing) <= missingMax {
			out = append(out, RecipeMatch{Recipe: r, Missing: missing})
		}
	}
	return out, nil
}
