package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

// maxReceipts caps one payments listing; a monthly plan with packs stays far below it.
const maxReceipts = 1000

var (
	errReceiptNotFound = errors.New("receipt not found")
)

// detailsError says which billing detail is invalid, in words for the customer.
type detailsError struct{ msg string }

func (e *detailsError) Error() string { return e.msg }

// PaymentReceipt is one payment of a workspace as its receipt shows it.
// Amounts are in the currency's subunits.
type PaymentReceipt struct {
	PaymentID string    `json:"paymentId"`
	ReceiptNo string    `json:"receiptNo"`
	Kind      string    `json:"kind"`
	Item      string    `json:"item"`
	Amount    int64     `json:"amount"`
	Refunded  int64     `json:"refunded"`
	Currency  string    `json:"currency"`
	PaidAt    time.Time `json:"paidAt"`
}

// Details is who a workspace's receipts are made out to.
type Details struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Country string `json:"country"`
	TaxID   string `json:"taxId"`
}

// Receipt is one payment and who it is billed to.
type Receipt struct {
	PaymentReceipt
	BilledTo Details `json:"billedTo"`
}

// receiptNumber formats a receipt number with the payment's year, e.g. UPL-2026-00042.
func receiptNumber(n int64, paidAt time.Time) string {
	return fmt.Sprintf("UPL-%d-%05d", paidAt.UTC().Year(), n)
}

// itemName is what a ledger item was, in the words the customer bought it under.
func itemName(kind, item string) string {
	if kind == "subscription" {
		if p, ok := plans.Get(item); ok {
			return "Uplumo " + p.Name + " plan"
		}
	}
	if p, ok := plans.GetPack(item); ok {
		return p.Name
	}
	return item
}

const receiptColumns = `payment_id, receipt_no, kind, item, amount, amount_refunded, currency, paid_at`

func scanReceipt(row pgx.Row) (PaymentReceipt, error) {
	var p PaymentReceipt
	var n int64
	if err := row.Scan(&p.PaymentID, &n, &p.Kind, &p.Item, &p.Amount, &p.Refunded, &p.Currency, &p.PaidAt); err != nil {
		return p, err
	}
	p.ReceiptNo = receiptNumber(n, p.PaidAt)
	p.Item = itemName(p.Kind, p.Item)
	return p, nil
}

// Payments lists the workspace's payments, newest first.
func (s *Service) Payments(ctx context.Context, orgID string) ([]PaymentReceipt, error) {
	rows, err := s.repo.db.Query(ctx, `SELECT `+receiptColumns+` FROM go_billing_payments
		WHERE organization_id=$1 ORDER BY paid_at DESC, receipt_no DESC LIMIT $2`, orgID, maxReceipts)
	if err != nil {
		return nil, fmt.Errorf("list payments of %s: %w", orgID, err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (PaymentReceipt, error) { return scanReceipt(row) })
	if err != nil {
		return nil, fmt.Errorf("list payments of %s: %w", orgID, err)
	}
	return out, nil
}

// Receipt returns one payment of the workspace with its billing details.
// A payment of another workspace is not found.
func (s *Service) Receipt(ctx context.Context, orgID, paymentID string) (Receipt, error) {
	p, err := scanReceipt(s.repo.db.QueryRow(ctx, `SELECT `+receiptColumns+` FROM go_billing_payments
		WHERE organization_id=$1 AND payment_id=$2`, orgID, paymentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, errReceiptNotFound
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("read payment %s: %w", paymentID, err)
	}
	details, err := s.BillingDetails(ctx, orgID)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{PaymentReceipt: p, BilledTo: details}, nil
}

// BillingDetails returns who the workspace's receipts are made out to; empty
// when it never said.
func (s *Service) BillingDetails(ctx context.Context, orgID string) (Details, error) {
	var d Details
	err := s.repo.db.QueryRow(ctx, `SELECT name, address, country, tax_id FROM go_billing_details WHERE organization_id=$1`, orgID).
		Scan(&d.Name, &d.Address, &d.Country, &d.TaxID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return d, fmt.Errorf("read billing details of %s: %w", orgID, err)
	}
	return d, nil
}

// SaveBillingDetails replaces the workspace's billing details. It returns
// a *detailsError saying what to fix when a field is invalid.
func (s *Service) SaveBillingDetails(ctx context.Context, orgID string, d Details) (Details, error) {
	d, err := cleanBillingDetails(d)
	if err != nil {
		return d, err
	}
	_, err = s.repo.db.Exec(ctx, `INSERT INTO go_billing_details(organization_id,name,address,country,tax_id) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (organization_id) DO UPDATE SET name=$2,address=$3,country=$4,tax_id=$5,updated_at=now()`,
		orgID, d.Name, d.Address, d.Country, d.TaxID)
	if err != nil {
		return d, fmt.Errorf("save billing details of %s: %w", orgID, err)
	}
	return d, nil
}

func cleanBillingDetails(d Details) (Details, error) {
	fields := []struct {
		value     *string
		label     string
		max       int
		required  bool
		multiline bool
	}{
		{&d.Name, "Name", 200, true, false},
		{&d.Address, "Address", 500, false, true},
		{&d.Country, "Country", 80, false, false},
		{&d.TaxID, "Tax ID", 40, false, false},
	}
	for _, f := range fields {
		*f.value = strings.TrimSpace(strings.ReplaceAll(*f.value, "\r\n", "\n"))
		v := *f.value
		switch {
		case f.required && v == "":
			return d, &detailsError{f.label + " is required."}
		case !utf8.ValidString(v):
			return d, &detailsError{f.label + " has invalid characters."}
		case utf8.RuneCountInString(v) > f.max:
			return d, &detailsError{fmt.Sprintf("%s must be %d characters or fewer.", f.label, f.max)}
		case strings.ContainsFunc(v, func(r rune) bool { return unicode.IsControl(r) && (!f.multiline || r != '\n') }):
			return d, &detailsError{f.label + " has invalid characters."}
		}
	}
	return d, nil
}
