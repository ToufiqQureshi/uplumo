-- +goose Up
-- Business (5 sites) sits between Growth ("pro") and Agency.
ALTER TABLE go_billing_subscriptions
    DROP CONSTRAINT go_billing_subscriptions_plan_check;
ALTER TABLE go_billing_subscriptions
    ADD CONSTRAINT go_billing_subscriptions_plan_check
    CHECK (plan IN ('pro', 'basic', 'business', 'agency'));

-- +goose Down
-- Subscription rows on the business plan must be moved before rolling back.
ALTER TABLE go_billing_subscriptions
    DROP CONSTRAINT go_billing_subscriptions_plan_check;
ALTER TABLE go_billing_subscriptions
    ADD CONSTRAINT go_billing_subscriptions_plan_check
    CHECK (plan IN ('pro', 'basic', 'agency'));
