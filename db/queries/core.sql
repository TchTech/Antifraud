-- name: LockGuard :one
SELECT * FROM account_guard WHERE account_id=$1 FOR UPDATE;
-- name: ChangeGuard :one
UPDATE account_guard SET revision=revision+1, held=$2, hold_until=CASE WHEN $2 THEN clock_timestamp()+interval '5 minutes' ELSE NULL END WHERE account_id=$1 RETURNING *;
-- name: GetProfile :one
SELECT snapshot FROM subscriber_profile WHERE account_id=$1;
-- name: SaveProfile :exec
INSERT INTO subscriber_profile(account_id,snapshot) VALUES ($1,$2) ON CONFLICT(account_id) DO UPDATE SET snapshot=EXCLUDED.snapshot;
-- name: Receipt :execrows
INSERT INTO event_receipt(source,event_id,payload_hash) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING;
-- name: ReceiptHash :one
SELECT payload_hash FROM event_receipt WHERE source=$1 AND event_id=$2;
-- name: AddRegistration :execrows
INSERT INTO sim_registration(event_id,account_id,source,schema_version,sim_id,cell_id,lat,lon,event_time) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(sim_id,cell_id,event_time) DO NOTHING;
-- name: RecentRegistrations :many
SELECT * FROM sim_registration WHERE account_id=$1 ORDER BY event_time DESC, event_id DESC LIMIT 2;
-- name: SaveOperation :execrows
INSERT INTO operation(operation_id,account_id,operation_type,parameters_hash,request) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING;
-- name: GetOperation :one
SELECT * FROM operation WHERE operation_id=$1;
-- name: LatestDecision :one
SELECT * FROM risk_decision WHERE account_id=$1 ORDER BY decided_at DESC,decision_id DESC LIMIT 1;
-- name: GetDecision :one
SELECT * FROM risk_decision WHERE decision_id=$1;
-- name: AddDecision :exec
INSERT INTO risk_decision(decision_id,account_id,operation_id,operation_type,parameters_hash,guard_revision,policy_version,result,decided_at,valid_until,evidence,supersedes_decision_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12);
-- name: AddConfirmation :exec
INSERT INTO operation_confirmation(confirmation_id,account_id,operation_id,operation_type,parameters_hash,guard_revision,expires_at,consumed_at) VALUES($1,$2,$3,$4,$5,$6,$7,NULL);
-- name: ConsumeConfirmation :execrows
WITH check_time AS MATERIALIZED (SELECT clock_timestamp() AS checked_at)
UPDATE operation_confirmation AS c SET consumed_at=t.checked_at FROM check_time AS t WHERE c.confirmation_id=$1 AND c.account_id=$2 AND c.operation_id=$3 AND c.operation_type=$4 AND c.parameters_hash=$5 AND c.guard_revision=$6 AND c.consumed_at IS NULL AND c.issued_at<=t.checked_at AND c.expires_at>t.checked_at;
-- name: Enqueue :exec
INSERT INTO execution_outbox(command_id,operation_id,account_id,decision_id) VALUES($1,$2,$3,$4) ON CONFLICT(operation_id) DO NOTHING;
-- name: Pending :many
SELECT * FROM execution_outbox WHERE delivered_at IS NULL ORDER BY created_at LIMIT 100;
-- name: GetCommand :one
SELECT * FROM execution_outbox WHERE command_id=$1;
-- name: GetExecution :one
SELECT * FROM execution_result WHERE command_id=$1;
-- name: SetService :exec
UPDATE service_state SET target=$2,changes=changes+1 WHERE account_id=$1;
-- name: GetService :one
SELECT * FROM service_state WHERE account_id=$1;
-- name: RecordExecution :exec
INSERT INTO execution_result(command_id,result) VALUES($1,$2);
-- name: Delivered :exec
UPDATE execution_outbox SET delivered_at=clock_timestamp() WHERE command_id=$1;
-- name: Audit :exec
INSERT INTO audit_event(account_id,operation_id,action,details) VALUES($1,$2,$3,$4);
-- name: Decisions :many
SELECT * FROM risk_decision WHERE account_id=$1 ORDER BY decided_at,decision_id;
-- name: Audits :many
SELECT * FROM audit_event WHERE account_id=$1 ORDER BY audit_id;
