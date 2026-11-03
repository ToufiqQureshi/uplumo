-- +goose Up
-- A subscription is billed in the currency of the Razorpay plan it was
-- created on, and a pack order in the currency it was created in. Every row
-- before dollar prices existed was in rupees. amount_paise holds the amount
-- in the currency's subunits (paise or cents).
ALTER TABLE go_billing_subscriptions
    ADD COLUMN currency text NOT NULL DEFAULT 'INR'
    CONSTRAINT go_billing_subscriptions_currency_check CHECK (currency IN ('INR', 'USD'));
ALTER TABLE go_billing_topups
    ADD COLUMN currency text NOT NULL DEFAULT 'INR'
    CONSTRAINT go_billing_topups_currency_check CHECK (currency IN ('INR', 'USD'));

-- +goose Down
ALTER TABLE go_billing_topups DROP COLUMN currency;
ALTER TABLE go_billing_subscriptions DROP COLUMN currency;
