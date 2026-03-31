// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

// Package store provides local SQLite persistence for redfin-pp-cli.
// Uses modernc.org/sqlite (pure Go, no CGO) for zero-dependency cross-compilation.
// FTS5 full-text search indexes are created for searchable content.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID returns true if the input looks like a UUID.
func IsUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// PropertyRecord represents a Redfin property listing.
type PropertyRecord struct {
	PropertyID   string
	ListingID    string
	Address      string
	City         string
	State        string
	Zip          string
	Price        int
	Beds         int
	Baths        float64
	Sqft         int
	LotSize      int
	YearBuilt    int
	PropertyType string
	Status       string
	DaysOnMarket int
	HOA          int
	Description  string
	URL          string
	Latitude     float64
	Longitude    float64
	Data         json.RawMessage
	SyncedAt     time.Time
	UpdatedAt    time.Time
}

// PropertyFilters controls filtering for ListProperties.
type PropertyFilters struct {
	City     string
	State    string
	Zip      string
	MinPrice int
	MaxPrice int
	MinBeds  int
	MaxBeds  int
	MinBaths float64
	MaxBaths float64
	MinSqft  int
	MaxSqft  int
	Status   string
	Limit    int
}

// ValuationRecord represents an AVM estimate snapshot.
type ValuationRecord struct {
	PropertyID   string
	Estimate     int
	EstimateLow  int
	EstimateHigh int
	Source       string
	CapturedAt   time.Time
}

// PriceHistoryRecord represents a price change event.
type PriceHistoryRecord struct {
	PropertyID string
	Price      int
	Status     string
	EventType  string
	CapturedAt time.Time
}

// PortfolioEntry represents a watched/owned property.
type PortfolioEntry struct {
	PropertyID string
	Label      string
	AddedAt    time.Time
	AlertBelow int
	AlertAbove int
	Notes      string
	Property   *PropertyRecord // joined
}

// TrendRecord represents a market trend data point.
type TrendRecord struct {
	RegionID   string
	Metric     string
	Value      float64
	Period     string
	CapturedAt time.Time
}

type Store struct {
	db   *sql.DB
	path string
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=ON&_temp_store=MEMORY&_mmap_size=268435456")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1)

	s := &Store{db: db, path: dbPath}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	migrations := []string{
		// Generic resources (kept from original)
		`CREATE TABLE IF NOT EXISTS resources (
			id TEXT PRIMARY KEY,
			resource_type TEXT NOT NULL,
			data JSON NOT NULL,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(resource_type)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_synced ON resources(synced_at)`,
		`CREATE TABLE IF NOT EXISTS sync_state (
			resource_type TEXT PRIMARY KEY,
			last_cursor TEXT,
			last_synced_at DATETIME,
			total_count INTEGER DEFAULT 0
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS resources_fts USING fts5(
			id, resource_type, content, tokenize='porter unicode61'
		)`,
		`CREATE TABLE IF NOT EXISTS stingray (
			id TEXT PRIMARY KEY,
			data JSON NOT NULL,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Properties (core entity)
		`CREATE TABLE IF NOT EXISTS properties (
			property_id TEXT PRIMARY KEY,
			listing_id TEXT,
			address TEXT,
			city TEXT,
			state TEXT,
			zip TEXT,
			price INTEGER,
			beds INTEGER,
			baths REAL,
			sqft INTEGER,
			lot_size INTEGER,
			year_built INTEGER,
			property_type TEXT,
			status TEXT,
			days_on_market INTEGER,
			hoa INTEGER,
			description TEXT,
			url TEXT,
			latitude REAL,
			longitude REAL,
			data JSON NOT NULL,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_properties_city ON properties(city)`,
		`CREATE INDEX IF NOT EXISTS idx_properties_zip ON properties(zip)`,
		`CREATE INDEX IF NOT EXISTS idx_properties_status ON properties(status)`,
		`CREATE INDEX IF NOT EXISTS idx_properties_price ON properties(price)`,

		// Property valuations (AVM tracking over time)
		`CREATE TABLE IF NOT EXISTS valuations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id TEXT NOT NULL,
			estimate INTEGER,
			estimate_low INTEGER,
			estimate_high INTEGER,
			source TEXT DEFAULT 'redfin',
			captured_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (property_id) REFERENCES properties(property_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_valuations_property ON valuations(property_id)`,

		// Price history (track changes over sync cycles)
		`CREATE TABLE IF NOT EXISTS price_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id TEXT NOT NULL,
			price INTEGER,
			status TEXT,
			event_type TEXT,
			captured_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (property_id) REFERENCES properties(property_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_price_history_property ON price_history(property_id)`,

		// Regions
		`CREATE TABLE IF NOT EXISTS regions (
			region_id TEXT PRIMARY KEY,
			region_type TEXT,
			name TEXT,
			market TEXT,
			data JSON NOT NULL,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Market trends
		`CREATE TABLE IF NOT EXISTS trends (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			region_id TEXT NOT NULL,
			metric TEXT NOT NULL,
			value REAL,
			period TEXT,
			captured_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (region_id) REFERENCES regions(region_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trends_region ON trends(region_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trends_metric ON trends(metric)`,

		// Portfolio (user's watched/owned properties)
		`CREATE TABLE IF NOT EXISTS portfolio (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id TEXT NOT NULL,
			label TEXT DEFAULT 'watching',
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			alert_below INTEGER,
			alert_above INTEGER,
			notes TEXT,
			FOREIGN KEY (property_id) REFERENCES properties(property_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_portfolio_property ON portfolio(property_id)`,

		// Scoring profiles
		`CREATE TABLE IF NOT EXISTS scoring_profiles (
			name TEXT PRIMARY KEY,
			config JSON NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// FTS for properties
		`CREATE VIRTUAL TABLE IF NOT EXISTS properties_fts USING fts5(
			property_id, address, city, state, zip, description, tokenize='porter unicode61'
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Generic resource methods (preserved from original)
// ---------------------------------------------------------------------------

func (s *Store) upsertGenericResourceTx(tx *sql.Tx, resourceType, id string, data json.RawMessage) error {
	_, err := tx.Exec(
		`INSERT INTO resources (id, resource_type, data, synced_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data, synced_at = excluded.synced_at, updated_at = excluded.updated_at`,
		id, resourceType, string(data), time.Now(), time.Now(),
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM resources_fts WHERE id = ?`, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: FTS index cleanup failed: %v\n", err)
	}

	_, err = tx.Exec(
		`INSERT INTO resources_fts (id, resource_type, content)
		 VALUES (?, ?, ?)`,
		id, resourceType, string(data),
	)
	if err != nil {
		// FTS insert failure is non-fatal
		fmt.Fprintf(os.Stderr, "warning: FTS index update failed: %v\n", err)
	}

	return nil
}

func (s *Store) Upsert(resourceType, id string, data json.RawMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.upsertGenericResourceTx(tx, resourceType, id, data); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) Get(resourceType, id string) (json.RawMessage, error) {
	var data string
	err := s.db.QueryRow(
		`SELECT data FROM resources WHERE resource_type = ? AND id = ?`,
		resourceType, id,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (s *Store) List(resourceType string, limit int) ([]json.RawMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT data FROM resources WHERE resource_type = ? ORDER BY updated_at DESC LIMIT ?`,
		resourceType, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		results = append(results, json.RawMessage(data))
	}
	return results, rows.Err()
}

func (s *Store) Search(query string, limit int) ([]json.RawMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT r.data FROM resources r
		 JOIN resources_fts f ON r.id = f.id
		 WHERE resources_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		results = append(results, json.RawMessage(data))
	}
	return results, rows.Err()
}

func extractObjectID(obj map[string]any) string {
	for _, key := range []string{"id", "ID", "uuid", "slug", "name"} {
		if v, ok := obj[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func lookupFieldValue(obj map[string]any, snakeKey string) any {
	if v, ok := obj[snakeKey]; ok {
		return v
	}
	parts := strings.Split(snakeKey, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	if v, ok := obj[strings.Join(parts, "")]; ok {
		return v
	}
	return nil
}

// UpsertBatch inserts or replaces multiple records in a single transaction.
// This is 10-100x faster than individual Upsert calls for bulk operations.
func (s *Store) UpsertBatch(resourceType string, items []json.RawMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting batch transaction: %w", err)
	}
	defer tx.Rollback()

	for _, item := range items {
		var obj map[string]any
		if err := json.Unmarshal(item, &obj); err != nil {
			continue
		}
		id := fmt.Sprintf("%v", lookupFieldValue(obj, "id"))
		if id == "" || id == "<nil>" {
			continue
		}

		_, err := tx.Exec(
			`INSERT OR REPLACE INTO resources (id, resource_type, data, synced_at, updated_at)
			 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			id, resourceType, string(item),
		)
		if err != nil {
			return fmt.Errorf("upserting %s/%s: %w", resourceType, id, err)
		}
	}

	return tx.Commit()
}

func (s *Store) SaveSyncState(resourceType, cursor string, count int) error {
	_, err := s.db.Exec(
		`INSERT INTO sync_state (resource_type, last_cursor, last_synced_at, total_count)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(resource_type) DO UPDATE SET last_cursor = excluded.last_cursor,
		 last_synced_at = excluded.last_synced_at, total_count = excluded.total_count`,
		resourceType, cursor, time.Now(), count,
	)
	return err
}

func (s *Store) GetSyncState(resourceType string) (cursor string, lastSynced time.Time, count int, err error) {
	err = s.db.QueryRow(
		`SELECT last_cursor, last_synced_at, total_count FROM sync_state WHERE resource_type = ?`,
		resourceType,
	).Scan(&cursor, &lastSynced, &count)
	if err == sql.ErrNoRows {
		return "", time.Time{}, 0, nil
	}
	return
}

// SaveSyncCursor stores the pagination cursor for a resource type.
func (s *Store) SaveSyncCursor(resourceType, cursor string) error {
	_, err := s.db.Exec(
		`INSERT INTO sync_state (resource_type, last_cursor, last_synced_at, total_count)
		 VALUES (?, ?, CURRENT_TIMESTAMP, 0)
		 ON CONFLICT(resource_type) DO UPDATE SET last_cursor = ?, last_synced_at = CURRENT_TIMESTAMP`,
		resourceType, cursor, cursor,
	)
	return err
}

// GetSyncCursor returns the last pagination cursor for a resource type.
func (s *Store) GetSyncCursor(resourceType string) string {
	var cursor sql.NullString
	s.db.QueryRow("SELECT last_cursor FROM sync_state WHERE resource_type = ?", resourceType).Scan(&cursor)
	if cursor.Valid {
		return cursor.String
	}
	return ""
}

// GetLastSyncedAt returns the last sync timestamp for a resource type.
func (s *Store) GetLastSyncedAt(resourceType string) string {
	var ts sql.NullString
	s.db.QueryRow("SELECT last_synced_at FROM sync_state WHERE resource_type = ?", resourceType).Scan(&ts)
	if ts.Valid {
		return ts.String
	}
	return ""
}

// ClearSyncCursors resets all sync state for a full resync.
func (s *Store) ClearSyncCursors() error {
	_, err := s.db.Exec("DELETE FROM sync_state")
	return err
}

// Query executes a raw SQL query and returns the rows.
// Used by workflow commands that need custom queries against the local store.
func (s *Store) Query(query string, args ...any) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

func (s *Store) Count(resourceType string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM resources WHERE resource_type = ?`,
		resourceType,
	).Scan(&count)
	return count, err
}

func (s *Store) Status() (map[string]int, error) {
	rows, err := s.db.Query(
		`SELECT resource_type, COUNT(*) FROM resources GROUP BY resource_type ORDER BY resource_type`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	status := make(map[string]int)
	for rows.Next() {
		var rt string
		var count int
		if err := rows.Scan(&rt, &count); err != nil {
			return nil, err
		}
		status[rt] = count
	}
	return status, rows.Err()
}

// ResolveByName resolves a human-readable name to a UUID from synced data.
// If the input is already a UUID, it is returned as-is.
// matchFields are JSON field names to search against (e.g., "name", "key", "email").
func (s *Store) ResolveByName(resourceType string, input string, matchFields ...string) (string, error) {
	if IsUUID(input) {
		return input, nil
	}

	var matches []string
	for _, field := range matchFields {
		query := fmt.Sprintf(
			`SELECT id FROM resources WHERE resource_type = ? AND LOWER(json_extract(data, '$.%s')) = LOWER(?)`,
			field,
		)
		rows, err := s.db.Query(query, resourceType, input)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				// Deduplicate
				found := false
				for _, m := range matches {
					if m == id {
						found = true
						break
					}
				}
				if !found {
					matches = append(matches, id)
				}
			}
		}
		rows.Close()
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%s %q not found in local store. Run 'sync' first, or use the UUID directly", resourceType, input)
	case 1:
		return matches[0], nil
	default:
		hint := matches[0]
		if len(matches) > 5 {
			hint = strings.Join(matches[:5], ", ") + "..."
		} else {
			hint = strings.Join(matches, ", ")
		}
		return "", fmt.Errorf("ambiguous: %q matches %d %s entries (%s). Use the exact UUID instead", input, len(matches), resourceType, hint)
	}
}

// ---------------------------------------------------------------------------
// Property methods
// ---------------------------------------------------------------------------

// UpsertProperty inserts or updates a property. If the price changed, a
// price_history record is automatically created.
func (s *Store) UpsertProperty(prop PropertyRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check for price change on existing property
	var oldPrice sql.NullInt64
	var oldStatus sql.NullString
	err = tx.QueryRow(`SELECT price, status FROM properties WHERE property_id = ?`, prop.PropertyID).Scan(&oldPrice, &oldStatus)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("checking existing property: %w", err)
	}

	now := time.Now()
	_, err = tx.Exec(
		`INSERT INTO properties (property_id, listing_id, address, city, state, zip, price, beds, baths, sqft, lot_size, year_built, property_type, status, days_on_market, hoa, description, url, latitude, longitude, data, synced_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(property_id) DO UPDATE SET
		   listing_id = excluded.listing_id,
		   address = excluded.address,
		   city = excluded.city,
		   state = excluded.state,
		   zip = excluded.zip,
		   price = excluded.price,
		   beds = excluded.beds,
		   baths = excluded.baths,
		   sqft = excluded.sqft,
		   lot_size = excluded.lot_size,
		   year_built = excluded.year_built,
		   property_type = excluded.property_type,
		   status = excluded.status,
		   days_on_market = excluded.days_on_market,
		   hoa = excluded.hoa,
		   description = excluded.description,
		   url = excluded.url,
		   latitude = excluded.latitude,
		   longitude = excluded.longitude,
		   data = excluded.data,
		   synced_at = excluded.synced_at,
		   updated_at = excluded.updated_at`,
		prop.PropertyID, prop.ListingID, prop.Address, prop.City, prop.State, prop.Zip,
		prop.Price, prop.Beds, prop.Baths, prop.Sqft, prop.LotSize, prop.YearBuilt,
		prop.PropertyType, prop.Status, prop.DaysOnMarket, prop.HOA,
		prop.Description, prop.URL, prop.Latitude, prop.Longitude,
		string(prop.Data), now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting property: %w", err)
	}

	// Auto-detect price or status changes and record in price_history
	if oldPrice.Valid {
		priceChanged := oldPrice.Int64 != int64(prop.Price)
		statusChanged := oldStatus.Valid && oldStatus.String != prop.Status
		if priceChanged || statusChanged {
			eventType := "price_change"
			if statusChanged && !priceChanged {
				eventType = "status_change"
			}
			_, err = tx.Exec(
				`INSERT INTO price_history (property_id, price, status, event_type) VALUES (?, ?, ?, ?)`,
				prop.PropertyID, prop.Price, prop.Status, eventType,
			)
			if err != nil {
				return fmt.Errorf("recording price history: %w", err)
			}
		}
	}

	// Update FTS index for properties
	_, _ = tx.Exec(`DELETE FROM properties_fts WHERE property_id = ?`, prop.PropertyID)
	_, _ = tx.Exec(
		`INSERT INTO properties_fts (property_id, address, city, state, zip, description) VALUES (?, ?, ?, ?, ?, ?)`,
		prop.PropertyID, prop.Address, prop.City, prop.State, prop.Zip, prop.Description,
	)

	return tx.Commit()
}

// GetProperty retrieves a single property by ID.
func (s *Store) GetProperty(propertyID string) (*PropertyRecord, error) {
	row := s.db.QueryRow(
		`SELECT property_id, listing_id, address, city, state, zip, price, beds, baths, sqft, lot_size, year_built, property_type, status, days_on_market, hoa, description, url, latitude, longitude, data, synced_at, updated_at
		 FROM properties WHERE property_id = ?`, propertyID,
	)
	p := &PropertyRecord{}
	err := row.Scan(
		&p.PropertyID, &p.ListingID, &p.Address, &p.City, &p.State, &p.Zip,
		&p.Price, &p.Beds, &p.Baths, &p.Sqft, &p.LotSize, &p.YearBuilt,
		&p.PropertyType, &p.Status, &p.DaysOnMarket, &p.HOA,
		&p.Description, &p.URL, &p.Latitude, &p.Longitude,
		&p.Data, &p.SyncedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// SearchProperties performs FTS search across properties.
func (s *Store) SearchProperties(query string, limit int) ([]PropertyRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT p.property_id, p.listing_id, p.address, p.city, p.state, p.zip, p.price, p.beds, p.baths, p.sqft, p.lot_size, p.year_built, p.property_type, p.status, p.days_on_market, p.hoa, p.description, p.url, p.latitude, p.longitude, p.data, p.synced_at, p.updated_at
		 FROM properties p
		 JOIN properties_fts f ON p.property_id = f.property_id
		 WHERE properties_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProperties(rows)
}

// ListProperties returns properties matching the given filters.
func (s *Store) ListProperties(filters PropertyFilters) ([]PropertyRecord, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 200
	}

	var conditions []string
	var args []any

	if filters.City != "" {
		conditions = append(conditions, "city = ?")
		args = append(args, filters.City)
	}
	if filters.State != "" {
		conditions = append(conditions, "state = ?")
		args = append(args, filters.State)
	}
	if filters.Zip != "" {
		conditions = append(conditions, "zip = ?")
		args = append(args, filters.Zip)
	}
	if filters.MinPrice > 0 {
		conditions = append(conditions, "price >= ?")
		args = append(args, filters.MinPrice)
	}
	if filters.MaxPrice > 0 {
		conditions = append(conditions, "price <= ?")
		args = append(args, filters.MaxPrice)
	}
	if filters.MinBeds > 0 {
		conditions = append(conditions, "beds >= ?")
		args = append(args, filters.MinBeds)
	}
	if filters.MaxBeds > 0 {
		conditions = append(conditions, "beds <= ?")
		args = append(args, filters.MaxBeds)
	}
	if filters.MinBaths > 0 {
		conditions = append(conditions, "baths >= ?")
		args = append(args, filters.MinBaths)
	}
	if filters.MaxBaths > 0 {
		conditions = append(conditions, "baths <= ?")
		args = append(args, filters.MaxBaths)
	}
	if filters.MinSqft > 0 {
		conditions = append(conditions, "sqft >= ?")
		args = append(args, filters.MinSqft)
	}
	if filters.MaxSqft > 0 {
		conditions = append(conditions, "sqft <= ?")
		args = append(args, filters.MaxSqft)
	}
	if filters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filters.Status)
	}

	q := `SELECT property_id, listing_id, address, city, state, zip, price, beds, baths, sqft, lot_size, year_built, property_type, status, days_on_market, hoa, description, url, latitude, longitude, data, synced_at, updated_at FROM properties`
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProperties(rows)
}

func scanProperties(rows *sql.Rows) ([]PropertyRecord, error) {
	var results []PropertyRecord
	for rows.Next() {
		var p PropertyRecord
		if err := rows.Scan(
			&p.PropertyID, &p.ListingID, &p.Address, &p.City, &p.State, &p.Zip,
			&p.Price, &p.Beds, &p.Baths, &p.Sqft, &p.LotSize, &p.YearBuilt,
			&p.PropertyType, &p.Status, &p.DaysOnMarket, &p.HOA,
			&p.Description, &p.URL, &p.Latitude, &p.Longitude,
			&p.Data, &p.SyncedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Valuation methods
// ---------------------------------------------------------------------------

// AddValuation records a new AVM estimate for a property.
func (s *Store) AddValuation(propertyID string, estimate, low, high int) error {
	_, err := s.db.Exec(
		`INSERT INTO valuations (property_id, estimate, estimate_low, estimate_high) VALUES (?, ?, ?, ?)`,
		propertyID, estimate, low, high,
	)
	return err
}

// GetValuationHistory returns all valuations for a property, newest first.
func (s *Store) GetValuationHistory(propertyID string) ([]ValuationRecord, error) {
	rows, err := s.db.Query(
		`SELECT property_id, estimate, estimate_low, estimate_high, source, captured_at
		 FROM valuations WHERE property_id = ? ORDER BY captured_at DESC`,
		propertyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ValuationRecord
	for rows.Next() {
		var v ValuationRecord
		if err := rows.Scan(&v.PropertyID, &v.Estimate, &v.EstimateLow, &v.EstimateHigh, &v.Source, &v.CapturedAt); err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Price history methods
// ---------------------------------------------------------------------------

// GetPriceHistory returns all price history events for a property, newest first.
func (s *Store) GetPriceHistory(propertyID string) ([]PriceHistoryRecord, error) {
	rows, err := s.db.Query(
		`SELECT property_id, price, status, event_type, captured_at
		 FROM price_history WHERE property_id = ? ORDER BY captured_at DESC`,
		propertyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PriceHistoryRecord
	for rows.Next() {
		var ph PriceHistoryRecord
		if err := rows.Scan(&ph.PropertyID, &ph.Price, &ph.Status, &ph.EventType, &ph.CapturedAt); err != nil {
			return nil, err
		}
		results = append(results, ph)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Portfolio methods
// ---------------------------------------------------------------------------

// AddToPortfolio adds a property to the user's portfolio.
func (s *Store) AddToPortfolio(propertyID, label string, alertBelow, alertAbove int, notes string) error {
	_, err := s.db.Exec(
		`INSERT INTO portfolio (property_id, label, alert_below, alert_above, notes) VALUES (?, ?, ?, ?, ?)`,
		propertyID, label, alertBelow, alertAbove, notes,
	)
	return err
}

// RemoveFromPortfolio removes a property from the portfolio.
func (s *Store) RemoveFromPortfolio(propertyID string) error {
	_, err := s.db.Exec(`DELETE FROM portfolio WHERE property_id = ?`, propertyID)
	return err
}

// ListPortfolio returns all portfolio entries with joined property data.
func (s *Store) ListPortfolio() ([]PortfolioEntry, error) {
	rows, err := s.db.Query(
		`SELECT po.property_id, po.label, po.added_at, po.alert_below, po.alert_above, po.notes,
		        p.property_id, p.listing_id, p.address, p.city, p.state, p.zip, p.price, p.beds, p.baths, p.sqft, p.lot_size, p.year_built, p.property_type, p.status, p.days_on_market, p.hoa, p.description, p.url, p.latitude, p.longitude, p.data, p.synced_at, p.updated_at
		 FROM portfolio po
		 LEFT JOIN properties p ON po.property_id = p.property_id
		 ORDER BY po.added_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PortfolioEntry
	for rows.Next() {
		var e PortfolioEntry
		var prop PropertyRecord
		var propID sql.NullString
		err := rows.Scan(
			&e.PropertyID, &e.Label, &e.AddedAt, &e.AlertBelow, &e.AlertAbove, &e.Notes,
			&propID, &prop.ListingID, &prop.Address, &prop.City, &prop.State, &prop.Zip,
			&prop.Price, &prop.Beds, &prop.Baths, &prop.Sqft, &prop.LotSize, &prop.YearBuilt,
			&prop.PropertyType, &prop.Status, &prop.DaysOnMarket, &prop.HOA,
			&prop.Description, &prop.URL, &prop.Latitude, &prop.Longitude,
			&prop.Data, &prop.SyncedAt, &prop.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if propID.Valid {
			prop.PropertyID = propID.String
			e.Property = &prop
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Region and trend methods
// ---------------------------------------------------------------------------

// UpsertRegion inserts or updates a region.
func (s *Store) UpsertRegion(regionID, regionType, name, market string, data json.RawMessage) error {
	_, err := s.db.Exec(
		`INSERT INTO regions (region_id, region_type, name, market, data, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(region_id) DO UPDATE SET
		   region_type = excluded.region_type,
		   name = excluded.name,
		   market = excluded.market,
		   data = excluded.data,
		   synced_at = excluded.synced_at`,
		regionID, regionType, name, market, string(data), time.Now(),
	)
	return err
}

// AddTrend records a market trend data point.
func (s *Store) AddTrend(regionID, metric string, value float64, period string) error {
	_, err := s.db.Exec(
		`INSERT INTO trends (region_id, metric, value, period) VALUES (?, ?, ?, ?)`,
		regionID, metric, value, period,
	)
	return err
}

// GetTrends returns trend data for a region, optionally filtered by metric.
func (s *Store) GetTrends(regionID string, metric string) ([]TrendRecord, error) {
	var rows *sql.Rows
	var err error
	if metric != "" {
		rows, err = s.db.Query(
			`SELECT region_id, metric, value, period, captured_at
			 FROM trends WHERE region_id = ? AND metric = ? ORDER BY captured_at DESC`,
			regionID, metric,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT region_id, metric, value, period, captured_at
			 FROM trends WHERE region_id = ? ORDER BY captured_at DESC`,
			regionID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TrendRecord
	for rows.Next() {
		var t TrendRecord
		if err := rows.Scan(&t.RegionID, &t.Metric, &t.Value, &t.Period, &t.CapturedAt); err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Scoring profile methods
// ---------------------------------------------------------------------------

// UpsertScoringProfile saves or updates a scoring profile configuration.
func (s *Store) UpsertScoringProfile(name string, config json.RawMessage) error {
	_, err := s.db.Exec(
		`INSERT INTO scoring_profiles (name, config)
		 VALUES (?, ?)
		 ON CONFLICT(name) DO UPDATE SET config = excluded.config`,
		name, string(config),
	)
	return err
}

// GetScoringProfile returns the configuration for a scoring profile.
func (s *Store) GetScoringProfile(name string) (json.RawMessage, error) {
	var data string
	err := s.db.QueryRow(`SELECT config FROM scoring_profiles WHERE name = ?`, name).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
