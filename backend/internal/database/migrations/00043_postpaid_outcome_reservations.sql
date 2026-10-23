-- +goose Up
-- A known postpaid outcome can start with no credits deducted. Persist it
-- before touching the balance so charging failures remain recoverable.
ALTER TABLE go_billing_credit_reservations
    DROP CONSTRAINT go_billing_credit_reservations_amount_credits_check,
    ADD CONSTRAINT go_billing_credit_reservations_amount_credits_check CHECK (amount_credits >= 0);

-- +goose Down
-- Retain support for recorded zero-hold outcomes on application rollback.
SELECT 1;
