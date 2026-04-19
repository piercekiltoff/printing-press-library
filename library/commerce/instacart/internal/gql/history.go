package gql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/commerce/instacart/internal/instacart"
	"github.com/mvanhorn/printing-press-library/library/commerce/instacart/internal/store"
)

// HistorySyncResult summarizes what one run of history sync wrote.
type HistorySyncResult struct {
	OrdersWritten         int            `json:"orders_written"`
	PurchasedItemsWritten int            `json:"purchased_items_written"`
	PerRetailer           map[string]int `json:"per_retailer_orders"`
}

// SyncHistory runs both history operations (CustomerOrderHistory and
// BuyItAgainPage), paginates through them, and upserts into the local
// SQLite store. Returns a summary and an error.
//
// maxOrders caps the first-run fetch (plan default: 50). Pass 0 for
// unlimited. sinceTime scopes to orders placed at or after the given
// time (used for incremental sync from `MostRecentOrderAt`); pass zero
// for a full fetch.
//
// Required operation hashes: BuyItAgainPage, CustomerOrderHistory. If
// either is empty in persisted_ops, this function surfaces a wrapped
// ErrHashMissing pointing at docs/history-ops-capture.md.
func SyncHistory(ctx context.Context, c *Client, maxOrders int, sinceTime time.Time) (*HistorySyncResult, error) {
	if err := requireHistoryHashes(c); err != nil {
		return nil, err
	}

	result := &HistorySyncResult{PerRetailer: map[string]int{}}

	// Pass 1: paginate CustomerOrderHistory, upsert orders + order_items
	// and seed purchased_items with first_purchased_at / last_purchased_at.
	var after string
	ordersFetched := 0
pagination:
	for {
		vars := map[string]any{"first": 25}
		if after != "" {
			vars["after"] = after
		}
		resp, err := c.Query(ctx, "CustomerOrderHistory", vars)
		if err != nil {
			return result, fmt.Errorf("CustomerOrderHistory page %d: %w", ordersFetched/25, err)
		}
		var parsed struct {
			Data struct {
				Orders struct {
					Edges    []orderEdge `json:"edges"`
					PageInfo pageInfo    `json:"pageInfo"`
				} `json:"orders"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.RawBody, &parsed); err != nil {
			return result, fmt.Errorf("decode CustomerOrderHistory: %w", err)
		}
		for _, e := range parsed.Data.Orders.Edges {
			if !sinceTime.IsZero() && !e.Node.PlacedAt.IsZero() && e.Node.PlacedAt.Before(sinceTime) {
				// We've crossed the incremental-sync boundary.
				break pagination
			}
			if err := writeOrder(c.Store, e.Node); err != nil {
				return result, err
			}
			result.OrdersWritten++
			result.PerRetailer[e.Node.RetailerSlug]++
			ordersFetched++
			if maxOrders > 0 && ordersFetched >= maxOrders {
				break pagination
			}
		}
		if !parsed.Data.Orders.PageInfo.HasNextPage {
			break
		}
		after = parsed.Data.Orders.PageInfo.EndCursor
		if after == "" {
			break
		}
	}

	// Pass 2: paginate BuyItAgainPage per-retailer to refresh the
	// purchased_items aggregate (brand, size, category, in_stock, last price).
	// We iterate the set of retailers we just saw orders at.
	for retailerSlug := range result.PerRetailer {
		written, err := syncBuyItAgainForRetailer(ctx, c, retailerSlug)
		if err != nil {
			return result, fmt.Errorf("BuyItAgainPage for %s: %w", retailerSlug, err)
		}
		result.PurchasedItemsWritten += written
	}

	return result, nil
}

// syncBuyItAgainForRetailer paginates BuyItAgainPage for one retailer
// and upserts purchased_items rows.
func syncBuyItAgainForRetailer(ctx context.Context, c *Client, retailerSlug string) (int, error) {
	var after string
	written := 0
	for {
		vars := map[string]any{"first": 50, "retailerSlug": retailerSlug}
		if after != "" {
			vars["after"] = after
		}
		resp, err := c.Query(ctx, "BuyItAgainPage", vars)
		if err != nil {
			return written, err
		}
		var parsed struct {
			Data struct {
				BuyItAgain struct {
					Edges    []biaEdge `json:"edges"`
					PageInfo pageInfo  `json:"pageInfo"`
				} `json:"buyItAgain"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.RawBody, &parsed); err != nil {
			return written, fmt.Errorf("decode BuyItAgainPage: %w", err)
		}
		for _, e := range parsed.Data.BuyItAgain.Edges {
			n := e.Node
			pi := store.PurchasedItem{
				ItemID:           n.ItemID,
				RetailerSlug:     n.RetailerSlug,
				ProductID:        n.ProductID,
				Name:             n.Name,
				Brand:            n.Brand,
				Size:             n.Size,
				Category:         n.Category,
				LastPurchasedAt:  n.LastPurchasedAt,
				FirstPurchasedAt: n.LastPurchasedAt, // best-effort; CustomerOrderHistory may overwrite
				PurchaseCount:    n.PurchaseCount,
				LastPriceCents:   n.LastPriceCents,
				LastInStock:      n.InStock,
			}
			if err := c.Store.UpsertPurchasedItem(pi, false); err != nil {
				return written, err
			}
			written++
		}
		if !parsed.Data.BuyItAgain.PageInfo.HasNextPage {
			break
		}
		after = parsed.Data.BuyItAgain.PageInfo.EndCursor
		if after == "" {
			break
		}
	}
	return written, nil
}

// writeOrder upserts one order + all its items, plus seeds purchased_items
// rows so queries work even before BuyItAgainPage has run.
func writeOrder(s *store.Store, n orderNode) error {
	if err := s.UpsertOrder(store.Order{
		OrderID:      n.ID,
		RetailerSlug: n.RetailerSlug,
		PlacedAt:     n.PlacedAt,
		Status:       n.Status,
		TotalCents:   n.TotalCents,
		ItemCount:    n.ItemCount,
	}); err != nil {
		return err
	}
	for _, it := range n.Items {
		if err := s.UpsertOrderItem(store.OrderItem{
			OrderID:      n.ID,
			ItemID:       it.ItemID,
			ProductID:    it.ProductID,
			Name:         it.Name,
			Quantity:     it.Quantity,
			QuantityType: it.QuantityType,
			PriceCents:   it.PriceCents,
		}); err != nil {
			return err
		}
		// Seed purchased_items; BuyItAgainPage pass will enrich with
		// brand/size/category. PurchaseCount=1 per order observation; later
		// sync runs avoid double-counting by NOT passing incrementCount.
		if err := s.UpsertPurchasedItem(store.PurchasedItem{
			ItemID:           it.ItemID,
			RetailerSlug:     n.RetailerSlug,
			ProductID:        it.ProductID,
			Name:             it.Name,
			LastPurchasedAt:  n.PlacedAt,
			FirstPurchasedAt: n.PlacedAt,
			PurchaseCount:    1,
			LastPriceCents:   it.PriceCents,
			LastInStock:      true,
		}, false); err != nil {
			return err
		}
	}
	return nil
}

// requireHistoryHashes returns an ErrHashMissing when either history op
// has an empty hash. Pointing the user at the capture doc is more useful
// than a generic GraphQL error.
func requireHistoryHashes(c *Client) error {
	if c.Store == nil {
		return fmt.Errorf("history sync needs a store")
	}
	for _, op := range instacart.HistoryOpNames() {
		h, _ := c.Store.LookupOp(op)
		if h == "" {
			return fmt.Errorf("%w: %s", ErrHashMissing, op)
		}
	}
	return nil
}

// ErrHashMissing marks a known operation whose hash has not yet been
// captured. CLI code can check errors.Is(err, ErrHashMissing) and emit
// a user-friendly pointer at docs/history-ops-capture.md.
var ErrHashMissing = fmt.Errorf("history GraphQL hash not captured yet")

// --- Response shapes for the history operations. Field names match the
//     GraphQL query text in internal/instacart/ops.go; if Instacart's
//     actual schema differs, adjust the query text and these structs in
//     the same commit. ---

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type orderEdge struct {
	Node orderNode `json:"node"`
}

type orderNode struct {
	ID           string          `json:"id"`
	PlacedAt     time.Time       `json:"placedAt"`
	Status       string          `json:"status"`
	RetailerSlug string          `json:"retailerSlug"`
	TotalCents   int64           `json:"totalCents"`
	ItemCount    int             `json:"itemCount"`
	Items        []orderItemNode `json:"items"`
}

type orderItemNode struct {
	ItemID       string  `json:"itemId"`
	ProductID    string  `json:"productId"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"`
	QuantityType string  `json:"quantityType"`
	PriceCents   int64   `json:"priceCents"`
}

type biaEdge struct {
	Node biaNode `json:"node"`
}

type biaNode struct {
	ItemID          string    `json:"itemId"`
	ProductID       string    `json:"productId"`
	Name            string    `json:"name"`
	Brand           string    `json:"brand"`
	Size            string    `json:"size"`
	Category        string    `json:"category"`
	RetailerSlug    string    `json:"retailerSlug"`
	LastPurchasedAt time.Time `json:"lastPurchasedAt"`
	PurchaseCount   int       `json:"purchaseCount"`
	LastPriceCents  int64     `json:"lastPriceCents"`
	InStock         bool      `json:"inStock"`
}
