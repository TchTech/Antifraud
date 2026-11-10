-- +goose Up
ALTER TABLE account_guard ADD CONSTRAINT revision_positive CHECK(revision>=0);
ALTER TABLE risk_decision ADD CONSTRAINT decision_window CHECK(valid_until>decided_at);
ALTER TABLE operation_confirmation ADD COLUMN issued_at timestamptz;
-- Existing demo confirmations used a fixed two-minute lifetime.
UPDATE operation_confirmation SET issued_at=LEAST(expires_at-interval '2 minutes',consumed_at);
ALTER TABLE operation_confirmation ALTER COLUMN issued_at SET DEFAULT clock_timestamp();
ALTER TABLE operation_confirmation ALTER COLUMN issued_at SET NOT NULL;
ALTER TABLE operation_confirmation ADD CONSTRAINT confirmation_window CHECK(expires_at>issued_at);
ALTER TABLE operation_confirmation ADD CONSTRAINT consumed_in_window CHECK(consumed_at IS NULL OR (consumed_at>=issued_at AND consumed_at<expires_at));
-- +goose Down
ALTER TABLE operation_confirmation DROP CONSTRAINT consumed_in_window;
ALTER TABLE operation_confirmation DROP CONSTRAINT confirmation_window;
ALTER TABLE operation_confirmation DROP COLUMN issued_at;
ALTER TABLE risk_decision DROP CONSTRAINT decision_window;
ALTER TABLE account_guard DROP CONSTRAINT revision_positive;
