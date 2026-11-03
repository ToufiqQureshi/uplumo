-- +goose Up
-- Every captured customer payment, kept for revenue reporting and accounting.
-- organization_id has no foreign key on purpose: erasing a workspace keeps its
-- payments (no email or name is stored here).
CREATE TABLE go_billing_payments (
 payment_id text PRIMARY KEY,
 organization_id text NOT NULL,
 kind text NOT NULL CHECK (kind IN ('subscription','topup')),
 item text NOT NULL,
 amount bigint NOT NULL CHECK (amount > 0),
 amount_refunded bigint NOT NULL DEFAULT 0 CHECK (amount_refunded >= 0 AND amount_refunded <= amount),
 currency text NOT NULL CHECK (currency IN ('INR','USD')),
 paid_at timestamptz NOT NULL,
 source text NOT NULL CHECK (source IN ('webhook','razorpay_sync','topup_backfill')),
 recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX go_billing_payments_paid_at ON go_billing_payments(paid_at);
CREATE INDEX go_billing_payments_organization ON go_billing_payments(organization_id);
ALTER TABLE go_billing_payments ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_billing_payments FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
 FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon','authenticated') LOOP
  EXECUTE format('REVOKE ALL ON TABLE go_billing_payments FROM %I',client_role);
 END LOOP;
END $$;
-- +goose StatementEnd
-- Credit packs already paid before this table existed.
INSERT INTO go_billing_payments(payment_id,organization_id,kind,item,amount,currency,paid_at,source)
SELECT payment_id,organization_id,'topup',pack_id,amount_paise,currency,created_at,'topup_backfill'
FROM go_billing_topups WHERE status='paid' AND payment_id IS NOT NULL AND amount_paise > 0
ON CONFLICT (payment_id) DO NOTHING;
-- +goose Down
DROP TABLE go_billing_payments;
