-- +goose Up
CREATE TABLE account_guard (account_id text PRIMARY KEY, revision bigint NOT NULL DEFAULT 0, held boolean NOT NULL DEFAULT false, hold_until timestamptz);
CREATE TABLE subscriber_profile (account_id text PRIMARY KEY REFERENCES account_guard, snapshot jsonb NOT NULL);
CREATE TABLE event_receipt (source text NOT NULL, event_id text NOT NULL, payload_hash text NOT NULL, PRIMARY KEY(source,event_id));
CREATE TABLE sim_registration (event_id text PRIMARY KEY, account_id text NOT NULL REFERENCES account_guard, source text NOT NULL, schema_version integer NOT NULL CHECK(schema_version>0), sim_id text NOT NULL, cell_id text NOT NULL, lat double precision NOT NULL, lon double precision NOT NULL, event_time timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE(sim_id,cell_id,event_time));
CREATE INDEX registration_account ON sim_registration(account_id,event_time DESC);
CREATE TABLE policy_version (version text PRIMARY KEY, definition jsonb NOT NULL);
INSERT INTO policy_version VALUES ('demo-v1','{"high":"hold","medium":"step_up","low":"allow","unknown":"hold","speedKmh":1000,"freshHours":24,"coolingMinutes":5,"permitMinutes":2}');
CREATE TABLE operation (operation_id text PRIMARY KEY, account_id text NOT NULL REFERENCES account_guard, operation_type text NOT NULL CHECK(operation_type='protect-call-forwarding'), parameters_hash text NOT NULL, request jsonb NOT NULL);
CREATE TABLE risk_decision (decision_id text PRIMARY KEY, account_id text NOT NULL REFERENCES account_guard, operation_id text NOT NULL REFERENCES operation, operation_type text NOT NULL, parameters_hash text NOT NULL, guard_revision bigint NOT NULL, policy_version text NOT NULL REFERENCES policy_version, result text NOT NULL CHECK(result IN ('allow','step_up','hold')), decided_at timestamptz NOT NULL, valid_until timestamptz NOT NULL, evidence jsonb NOT NULL, supersedes_decision_id text REFERENCES risk_decision);
CREATE INDEX decision_account ON risk_decision(account_id,decided_at DESC);
CREATE TABLE operation_confirmation (confirmation_id text PRIMARY KEY, account_id text NOT NULL REFERENCES account_guard, operation_id text NOT NULL REFERENCES operation, operation_type text NOT NULL, parameters_hash text NOT NULL, guard_revision bigint NOT NULL, expires_at timestamptz NOT NULL, consumed_at timestamptz);
CREATE TABLE execution_outbox (command_id text PRIMARY KEY, operation_id text NOT NULL UNIQUE REFERENCES operation, account_id text NOT NULL REFERENCES account_guard, decision_id text NOT NULL REFERENCES risk_decision, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), delivered_at timestamptz);
CREATE TABLE service_state (account_id text PRIMARY KEY REFERENCES account_guard, target text NOT NULL DEFAULT 'disabled', changes bigint NOT NULL DEFAULT 0);
CREATE TABLE execution_result (command_id text PRIMARY KEY REFERENCES execution_outbox, result text NOT NULL CHECK(result IN ('executed','rejected')), recorded_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE audit_event (audit_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, account_id text NOT NULL, operation_id text NOT NULL, action text NOT NULL, details jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp());
-- +goose StatementBegin
CREATE FUNCTION immutable_history() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'append-only history'; END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_decision BEFORE UPDATE OR DELETE ON risk_decision FOR EACH ROW EXECUTE FUNCTION immutable_history();
CREATE TRIGGER immutable_policy BEFORE UPDATE OR DELETE ON policy_version FOR EACH ROW EXECUTE FUNCTION immutable_history();
CREATE TRIGGER immutable_audit BEFORE UPDATE OR DELETE ON audit_event FOR EACH ROW EXECUTE FUNCTION immutable_history();
CREATE TRIGGER immutable_execution BEFORE UPDATE OR DELETE ON execution_result FOR EACH ROW EXECUTE FUNCTION immutable_history();
INSERT INTO account_guard(account_id) VALUES ('demo-fraud'),('demo-legal'),('demo-clean');
INSERT INTO service_state(account_id) SELECT account_id FROM account_guard;
-- Runtime has no history UPDATE/DELETE/TRUNCATE or schema ownership.
-- +goose StatementBegin
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='antifraud_runtime') THEN
    CREATE ROLE antifraud_runtime NOLOGIN;
  END IF;
  EXECUTE format('GRANT USAGE ON SCHEMA %I TO antifraud_runtime',current_schema());
END $$;
-- +goose StatementEnd
GRANT SELECT ON account_guard,subscriber_profile,event_receipt,sim_registration,policy_version,operation,risk_decision,operation_confirmation,execution_outbox,service_state,execution_result,audit_event TO antifraud_runtime;
GRANT INSERT ON subscriber_profile,event_receipt,sim_registration,operation,risk_decision,operation_confirmation,execution_outbox,execution_result,audit_event TO antifraud_runtime;
GRANT UPDATE ON account_guard,subscriber_profile,operation_confirmation,execution_outbox,service_state TO antifraud_runtime;
GRANT USAGE ON SEQUENCE audit_event_audit_id_seq TO antifraud_runtime;
-- +goose Down
DROP TABLE audit_event, execution_result, service_state, execution_outbox, operation_confirmation, risk_decision, operation, policy_version, sim_registration, event_receipt, subscriber_profile, account_guard;
DROP FUNCTION immutable_history();
