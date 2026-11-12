-- +goose Up
-- Receipt numbers: one per payment, in payment order, never reused or changed.
ALTER TABLE go_billing_payments ADD COLUMN receipt_no bigint;
UPDATE go_billing_payments p SET receipt_no = n.rn
FROM (SELECT payment_id, row_number() OVER (ORDER BY paid_at, payment_id) AS rn FROM go_billing_payments) n
WHERE p.payment_id = n.payment_id;
CREATE SEQUENCE go_billing_receipt_no_seq OWNED BY go_billing_payments.receipt_no;
SELECT setval('go_billing_receipt_no_seq', coalesce((SELECT max(receipt_no) FROM go_billing_payments), 0) + 1, false);
ALTER TABLE go_billing_payments
 ALTER COLUMN receipt_no SET DEFAULT nextval('go_billing_receipt_no_seq'),
 ALTER COLUMN receipt_no SET NOT NULL,
 ADD CONSTRAINT go_billing_payments_receipt_no_key UNIQUE (receipt_no);

-- Who a workspace's receipts are made out to. Personal data: it is erased
-- with the workspace, unlike the payments themselves.
CREATE TABLE go_billing_details (
 organization_id text PRIMARY KEY REFERENCES organization(id) ON DELETE CASCADE,
 name text NOT NULL,
 address text NOT NULL DEFAULT '',
 country text NOT NULL DEFAULT '',
 tax_id text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE go_billing_details ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_billing_details FROM PUBLIC;
REVOKE ALL ON SEQUENCE go_billing_receipt_no_seq FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
 FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon','authenticated') LOOP
  EXECUTE format('REVOKE ALL ON TABLE go_billing_details FROM %I',client_role);
  EXECUTE format('REVOKE ALL ON SEQUENCE go_billing_receipt_no_seq FROM %I',client_role);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP TABLE go_billing_details;
ALTER TABLE go_billing_payments DROP COLUMN receipt_no;
