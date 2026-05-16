package memberships

import (
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// BillPreviewItem is one line of a bill-preview output.
type BillPreviewItem struct {
	SkuID       int64   `json:"sku_id"`
	SkuType     string  `json:"sku_type"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	LineTotal   float64 `json:"line_total"`
	IsAddOn     bool    `json:"is_add_on"`
}

// BillPreviewResult is the resolved bill-walk output for a single membership.
type BillPreviewResult struct {
	MembershipID     int64             `json:"membership_id"`
	MembershipTypeID int64             `json:"membership_type_id"`
	NextBillDate     string            `json:"next_bill_date"`
	BillingFrequency string            `json:"billing_frequency"`
	BillingPrice     float64           `json:"billing_price"`
	SalePrice        float64           `json:"sale_price"`
	RenewalPrice     float64           `json:"renewal_price"`
	InvoiceTemplate  string            `json:"invoice_template"`
	Items            []BillPreviewItem `json:"items"`
	Total            float64           `json:"total"`
	Notes            []string          `json:"notes,omitempty"`
}

// BillPreview walks one membership's bill chain: membership →
// membership-type's durationBilling → invoice-template items, and returns
// the next bill date plus a per-line breakdown. The walk surfaces gaps
// (no matching durationBilling row, no invoice template) as Notes rather
// than errors so a partially-configured membership still produces useful
// output — the user sees what's missing instead of an opaque "no data".
func BillPreview(db *store.Store, membershipID int64) (BillPreviewResult, error) {
	memberships, err := LoadMemberships(db)
	if err != nil {
		return BillPreviewResult{}, err
	}
	var target *Membership
	for i := range memberships {
		if memberships[i].ID == membershipID {
			target = &memberships[i]
			break
		}
	}
	if target == nil {
		return BillPreviewResult{}, fmt.Errorf("membership %d not found in local store; run 'sync' first or check the ID", membershipID)
	}
	types, err := LoadMembershipTypes(db)
	if err != nil {
		return BillPreviewResult{}, err
	}
	templates, err := LoadInvoiceTemplates(db)
	if err != nil {
		return BillPreviewResult{}, err
	}

	result := BillPreviewResult{
		MembershipID:     target.ID,
		MembershipTypeID: target.MembershipTypeID,
		BillingFrequency: target.BillingFrequency,
	}
	if target.NextScheduledBillDate != nil {
		result.NextBillDate = *target.NextScheduledBillDate
	} else {
		result.Notes = append(result.Notes, "membership has no nextScheduledBillDate")
	}

	mt, ok := types[target.MembershipTypeID]
	if !ok {
		result.Notes = append(result.Notes, fmt.Sprintf("membership-type %d not in local store; run 'sync' for membership-types", target.MembershipTypeID))
		return result, nil
	}
	entry, ok := lookupDurationBilling(mt.DurationBilling, target.Duration, target.BillingFrequency)
	if !ok {
		result.Notes = append(result.Notes, "no matching durationBilling entry on membership-type")
	} else {
		result.BillingPrice = entry.BillingPrice
		result.SalePrice = entry.SalePrice
		result.RenewalPrice = entry.RenewalPrice
	}

	templateID := mt.BillingTemplateID
	if templateID == nil {
		result.Notes = append(result.Notes, "membership-type has no billingTemplateId")
		return result, nil
	}
	tpl, ok := templates[*templateID]
	if !ok {
		result.Notes = append(result.Notes, fmt.Sprintf("invoice-template %d not in local store; run 'sync' for invoice-templates", *templateID))
		return result, nil
	}
	result.InvoiceTemplate = tpl.Name
	var total float64
	for _, it := range tpl.Items {
		line := round2(it.Quantity * it.UnitPrice)
		total += line
		result.Items = append(result.Items, BillPreviewItem{
			SkuID: it.SkuID, SkuType: it.SkuType, Description: it.Description,
			Quantity: it.Quantity, UnitPrice: round2(it.UnitPrice), LineTotal: line, IsAddOn: it.IsAddOn,
		})
	}
	result.Total = round2(total)
	return result, nil
}

// _ silences unused import on partial builds.
var _ = time.Now
