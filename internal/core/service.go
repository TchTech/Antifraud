package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	api "antifraud/internal/api/riskapi"
	"antifraud/internal/policy"
	"antifraud/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConflict = errors.New("operation binding, revision or confirmation rejected")

const OperationType = "protect-call-forwarding"

type Service struct{ Pool *pgxpool.Pool }

func raw(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func hash(v any) string                 { h := sha256.Sum256(raw(v)); return hex.EncodeToString(h[:]) }
func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func (s *Service) transaction(ctx context.Context, account string, fn func(*store.Queries, store.AccountGuard) error) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SET LOCAL synchronous_commit=on"); e != nil {
		return e
	}
	q := store.New(tx)
	g, e := q.LockGuard(ctx, account)
	if e != nil {
		return e
	}
	if g.Held && g.HoldUntil.Valid && time.Now().After(g.HoldUntil.Time) {
		g, e = q.ChangeGuard(ctx, store.ChangeGuardParams{AccountID: account, Held: false})
		if e != nil {
			return e
		}
		if e = audit(ctx, q, account, "", "cooling_expired", map[string]any{"revision": g.Revision}); e != nil {
			return e
		}
	}
	if e = fn(q, g); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func audit(ctx context.Context, q *store.Queries, account, op, action string, v any) error {
	return q.Audit(ctx, store.AuditParams{AccountID: account, OperationID: op, Action: action, Details: raw(v)})
}
func (s *Service) Evaluate(ctx context.Context, r api.Evaluation) (api.Decision, error) {
	var d api.Decision
	e := s.transaction(ctx, string(r.AccountId), func(q *store.Queries, g store.AccountGuard) error {
		var e error
		d, e = s.evaluate(ctx, q, g, r, "")
		return e
	})
	return d, e
}
func (s *Service) evaluate(ctx context.Context, q *store.Queries, g store.AccountGuard, r api.Evaluation, supersedes string) (api.Decision, error) {
	op := r.OperationId.String()
	ph := hash(map[string]string{"target": string(r.Target)})
	_, e := q.SaveOperation(ctx, store.SaveOperationParams{OperationID: op, AccountID: g.AccountID, OperationType: OperationType, ParametersHash: ph, Request: raw(r)})
	if e != nil {
		return api.Decision{}, e
	}
	existing, e := q.GetOperation(ctx, op)
	if e != nil {
		return api.Decision{}, e
	}
	if existing.AccountID != g.AccountID || existing.ParametersHash != ph {
		return api.Decision{}, ErrConflict
	}
	var p policy.Profile
	b, e := q.GetProfile(ctx, g.AccountID)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return api.Decision{}, e
	}
	if len(b) > 0 {
		if e = json.Unmarshal(b, &p); e != nil {
			return api.Decision{}, e
		}
	}
	f := policy.Calculate(p, string(r.Device), string(r.Attestation), time.Now().UTC())
	result, reason := policy.Decide(f)
	f.Reason = reason
	if g.Held {
		result = "hold"
		f.Reason = "active_cooling"
	}
	if result == "hold" && !g.Held {
		g, e = q.ChangeGuard(ctx, store.ChangeGuardParams{AccountID: g.AccountID, Held: true})
		if e != nil {
			return api.Decision{}, e
		}
	}
	d, e := saveDecision(ctx, q, g, existing, result, f, supersedes)
	if e != nil {
		return d, e
	}
	if result == "step_up" {
		id := uuid.NewString()
		d.ConfirmationId = &id
		e = q.AddConfirmation(ctx, store.AddConfirmationParams{ConfirmationID: id, AccountID: g.AccountID, OperationID: op, OperationType: OperationType, ParametersHash: ph, GuardRevision: g.Revision, ExpiresAt: ts(time.Now().Add(2 * time.Minute))})
	}
	return d, e
}
func saveDecision(ctx context.Context, q *store.Queries, g store.AccountGuard, op store.Operation, result string, f policy.Features, supersedes string) (api.Decision, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	e := q.AddDecision(ctx, store.AddDecisionParams{DecisionID: id, AccountID: g.AccountID, OperationID: op.OperationID, OperationType: op.OperationType, ParametersHash: op.ParametersHash, GuardRevision: g.Revision, PolicyVersion: policy.Version, Result: result, DecidedAt: ts(now), ValidUntil: ts(now.Add(2 * time.Minute)), Evidence: raw(f), SupersedesDecisionID: pgtype.Text{String: supersedes, Valid: supersedes != ""}})
	d := api.Decision{DecisionId: id, OperationId: op.OperationID, Result: api.DecisionResult(result), PolicyVersion: policy.Version}
	if e != nil {
		return d, e
	}
	if e = audit(ctx, q, g.AccountID, op.OperationID, "decision", map[string]any{"decisionId": id, "result": result, "evidence": f}); e != nil {
		return d, e
	}
	if result == "allow" {
		e = q.Enqueue(ctx, store.EnqueueParams{CommandID: uuid.NewString(), OperationID: op.OperationID, AccountID: g.AccountID, DecisionID: id})
	}
	return d, e
}
func (s *Service) Confirm(ctx context.Context, r api.Confirmation) (api.Decision, error) {
	var d api.Decision
	e := s.transaction(ctx, string(r.AccountId), func(q *store.Queries, g store.AccountGuard) error {
		if string(r.Proof) != "demo-approved" || g.Held {
			return ErrConflict
		}
		op, e := q.GetOperation(ctx, r.OperationId.String())
		if e != nil {
			return e
		}
		if op.AccountID != g.AccountID {
			return ErrConflict
		}
		last, e := q.LatestDecision(ctx, g.AccountID)
		if e != nil {
			return e
		}
		if last.OperationID != op.OperationID || last.Result != "step_up" || last.GuardRevision != g.Revision || time.Now().After(last.ValidUntil.Time) {
			return ErrConflict
		}
		n, e := q.ConsumeConfirmation(ctx, store.ConsumeConfirmationParams{ConfirmationID: r.ConfirmationId.String(), AccountID: g.AccountID, OperationID: op.OperationID, OperationType: op.OperationType, ParametersHash: op.ParametersHash, GuardRevision: g.Revision})
		if e != nil {
			return e
		}
		if n != 1 {
			return ErrConflict
		}
		var f policy.Features
		if e = json.Unmarshal(last.Evidence, &f); e != nil {
			return e
		}
		f.Confirmation = "demo-approved"
		f.Reason = "test_confirmation_consumed"
		d, e = saveDecision(ctx, q, g, op, "allow", f, last.DecisionID)
		return e
	})
	return d, e
}

// Hold shares the serialization point with evaluation, ingest and actual execution.
func (s *Service) Hold(ctx context.Context, account string) error {
	return s.transaction(ctx, account, func(q *store.Queries, g store.AccountGuard) error {
		_, e := q.ChangeGuard(ctx, store.ChangeGuardParams{AccountID: account, Held: true})
		if e != nil {
			return e
		}
		return audit(ctx, q, account, "", "hold", map[string]string{"reason": "test restriction"})
	})
}
func (s *Service) Deliver(ctx context.Context, id string) error {
	command, e := store.New(s.Pool).GetCommand(ctx, id)
	if e != nil {
		return e
	}
	return s.transaction(ctx, command.AccountID, func(q *store.Queries, g store.AccountGuard) error {
		_, e := q.GetExecution(ctx, id)
		if e == nil {
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		d, e := q.GetDecision(ctx, command.DecisionID)
		if e != nil {
			return e
		}
		op, e := q.GetOperation(ctx, command.OperationID)
		if e != nil {
			return e
		}
		result := "rejected"
		if !g.Held && d.Result == "allow" && d.AccountID == g.AccountID && d.OperationID == op.OperationID && d.OperationType == op.OperationType && d.ParametersHash == op.ParametersHash && d.GuardRevision == g.Revision && time.Now().Before(d.ValidUntil.Time) {
			var r api.Evaluation
			if e = json.Unmarshal(op.Request, &r); e != nil {
				return e
			}
			if hash(map[string]string{"target": string(r.Target)}) != d.ParametersHash {
				return ErrConflict
			}
			if e = q.SetService(ctx, store.SetServiceParams{AccountID: g.AccountID, Target: string(r.Target)}); e != nil {
				return e
			}
			result = "executed"
		}
		if e = q.RecordExecution(ctx, store.RecordExecutionParams{CommandID: id, Result: result}); e != nil {
			return e
		}
		if e = audit(ctx, q, g.AccountID, op.OperationID, "execution", map[string]string{"commandId": id, "decisionId": d.DecisionID, "result": result}); e != nil {
			return e
		}
		return q.Delivered(ctx, id)
	})
}
func (s *Service) Drain(ctx context.Context) error {
	cmds, e := store.New(s.Pool).Pending(ctx)
	if e != nil {
		return e
	}
	for _, c := range cmds {
		if e = s.Deliver(ctx, c.CommandID); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) State(ctx context.Context, account string) (api.State, error) {
	q := store.New(s.Pool)
	v, e := q.GetService(ctx, account)
	if e != nil {
		return api.State{}, e
	}
	ds, e := q.Decisions(ctx, account)
	if e != nil {
		return api.State{}, e
	}
	as, e := q.Audits(ctx, account)
	if e != nil {
		return api.State{}, e
	}
	maps := func(v any) []map[string]interface{} {
		var out []map[string]interface{}
		if e := json.Unmarshal(raw(v), &out); e != nil {
			panic(e)
		}
		return out
	}
	return api.State{Target: v.Target, Changes: int(v.Changes), Decisions: maps(ds), Audit: maps(as)}, nil
}
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL required (isolated synthetic database)")
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		return nil, e
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, e := conn.Exec(ctx, "SET ROLE antifraud_runtime")
		return e
	}
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return p, nil
}
