package memberships

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// membership_status_snapshots is the append-only log of (status,
// followUpStatus, nextScheduledBillDate, to) per active membership at a
// point in time. The ServiceTitan API exposes a per-membership
// status-changes endpoint but no bulk history; this table is the single-
// query backbone for after-the-fact "did a renewal slip into cancelled"
// trends, and for the complete-command's local-snapshot refresh path that
// keeps `overdue-events` honest in the same shell session.
const createStatusSnapshots = `
CREATE TABLE IF NOT EXISTS membership_status_snapshots (
	snapshot_at              TEXT    NOT NULL,
	membership_id            INTEGER NOT NULL,
	customer_id              INTEGER,
	membership_type_id       INTEGER,
	status                   TEXT,
	follow_up_status         TEXT,
	active                   INTEGER,
	next_scheduled_bill_date TEXT,
	to_date                  TEXT,
	modified_on              TEXT,
	PRIMARY KEY (snapshot_at, membership_id)
);`

// EnsureStatusSnapshots creates the membership_status_snapshots table if it
// does not exist. Safe to call on every command invocation.
func EnsureStatusSnapshots(db *store.Store) error {
	if _, err := db.DB().Exec(createStatusSnapshots); err != nil {
		return fmt.Errorf("creating membership_status_snapshots: %w", err)
	}
	return nil
}

// SnapshotMembershipStatus writes one row per active membership to
// membership_status_snapshots, but only for memberships whose status,
// follow-up status, or next-bill date changed since their most recent prior
// row. The table stays a true change log instead of growing by the full
// membership set on every call. Returns the number of rows written and the
// number considered. Callers refresh the snapshot after writes (e.g. the
// complete command after marking an event done) so trend queries reflect
// the latest state.
func SnapshotMembershipStatus(db *store.Store) (written, considered int, err error) {
	if err = EnsureStatusSnapshots(db); err != nil {
		return 0, 0, err
	}
	prior, err := latestStatusSnapshotByID(db)
	if err != nil {
		return 0, 0, err
	}
	memberships, err := LoadMemberships(db)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := db.DB().Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("begin snapshot tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO membership_status_snapshots
		(snapshot_at, membership_id, customer_id, membership_type_id, status,
		 follow_up_status, active, next_scheduled_bill_date, to_date, modified_on)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, 0, fmt.Errorf("prepare status snapshot insert: %w", err)
	}
	defer stmt.Close()

	for _, m := range memberships {
		if !m.Active {
			continue
		}
		considered++
		next := ""
		if m.NextScheduledBillDate != nil {
			next = *m.NextScheduledBillDate
		}
		toDate := ""
		if m.To != nil {
			toDate = *m.To
		}
		if prev, ok := prior[m.ID]; ok &&
			prev.status == m.Status &&
			prev.followUpStatus == m.FollowUpStatus &&
			prev.nextScheduledBillDate == next &&
			prev.toDate == toDate {
			continue
		}
		activeInt := 0
		if m.Active {
			activeInt = 1
		}
		if _, err := stmt.Exec(now, m.ID, m.CustomerID, m.MembershipTypeID,
			m.Status, m.FollowUpStatus, activeInt, next, toDate, m.ModifiedOn); err != nil {
			return 0, 0, fmt.Errorf("insert status snapshot: %w", err)
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit status snapshot: %w", err)
	}
	return written, considered, nil
}

type statusSnap struct {
	status                string
	followUpStatus        string
	nextScheduledBillDate string
	toDate                string
}

func latestStatusSnapshotByID(db *store.Store) (map[int64]statusSnap, error) {
	rows, err := db.DB().Query(`
		SELECT s.membership_id, s.status, COALESCE(s.follow_up_status,''),
		       COALESCE(s.next_scheduled_bill_date,''), COALESCE(s.to_date,'')
		FROM membership_status_snapshots s
		JOIN (
			SELECT membership_id, MAX(snapshot_at) AS mx
			FROM membership_status_snapshots GROUP BY membership_id
		) latest
		ON s.membership_id = latest.membership_id AND s.snapshot_at = latest.mx`)
	if err != nil {
		return nil, fmt.Errorf("query latest status snapshots: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]statusSnap)
	for rows.Next() {
		var id int64
		var sn statusSnap
		if err := rows.Scan(&id, &sn.status, &sn.followUpStatus, &sn.nextScheduledBillDate, &sn.toDate); err != nil {
			return nil, fmt.Errorf("scan status snapshot: %w", err)
		}
		out[id] = sn
	}
	return out, rows.Err()
}

// StatusHistoryRow is one row of a single membership's status history.
type StatusHistoryRow struct {
	SnapshotAt            string `json:"snapshot_at"`
	Status                string `json:"status"`
	FollowUpStatus        string `json:"follow_up_status"`
	Active                bool   `json:"active"`
	NextScheduledBillDate string `json:"next_scheduled_bill_date"`
	ToDate                string `json:"to_date"`
}

// MembershipStatusHistory returns the snapshot rows for one membership,
// ordered oldest first. Callers wire this into ad-hoc inspection paths
// (currently used by the test suite to assert idempotence).
func MembershipStatusHistory(db *store.Store, id int64) ([]StatusHistoryRow, error) {
	if err := EnsureStatusSnapshots(db); err != nil {
		return nil, err
	}
	rows, err := db.DB().Query(`
		SELECT snapshot_at, COALESCE(status,''), COALESCE(follow_up_status,''),
		       COALESCE(active,0), COALESCE(next_scheduled_bill_date,''), COALESCE(to_date,'')
		FROM membership_status_snapshots
		WHERE membership_id = ?
		ORDER BY snapshot_at`, id)
	if err != nil {
		return nil, fmt.Errorf("query status history: %w", err)
	}
	defer rows.Close()
	var out []StatusHistoryRow
	for rows.Next() {
		var r StatusHistoryRow
		var activeInt int
		if err := rows.Scan(&r.SnapshotAt, &r.Status, &r.FollowUpStatus, &activeInt, &r.NextScheduledBillDate, &r.ToDate); err != nil {
			return nil, fmt.Errorf("scan status history: %w", err)
		}
		r.Active = activeInt != 0
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// StatusSnapshotRows returns the total number of rows in
// membership_status_snapshots. Used by the health rollup.
func StatusSnapshotRows(db *store.Store) (int, error) {
	if err := EnsureStatusSnapshots(db); err != nil {
		return 0, err
	}
	var n int
	err := db.DB().QueryRow(`SELECT COUNT(*) FROM membership_status_snapshots`).Scan(&n)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("counting status snapshots: %w", err)
	}
	return n, nil
}
