-- +goose Up
ALTER TABLE go_billing_credit_reservations
    ADD COLUMN provider_outcome_pending boolean NOT NULL DEFAULT false,
    ADD COLUMN settlement_credits bigint CHECK (settlement_credits >= 0),
    ADD COLUMN provider_cost_usd numeric(12, 6) CHECK (provider_cost_usd >= 0);

-- Existing feature holds may already have reached a provider. Age alone
-- cannot establish that refunding them is safe.
UPDATE go_billing_credit_reservations SET provider_outcome_pending = true
WHERE status = 'reserved' AND feature <> '';

-- +goose Down
ALTER TABLE go_billing_credit_reservations
    DROP COLUMN provider_outcome_pending,
    DROP COLUMN settlement_credits,
    DROP COLUMN provider_cost_usd;
