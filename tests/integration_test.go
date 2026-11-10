//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	api "antifraud/internal/api/riskapi"
	"antifraud/internal/core"
	"antifraud/internal/store"
	"antifraud/internal/transport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var ctx = context.Background()

func setup(t *testing.T) *core.Service {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL required; use isolated PostgreSQL, never a real database")
	}
	admin, e := pgxpool.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + uuid.NewString()[:8]
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()
	if e = goose.SetDialect("postgres"); e != nil {
		t.Fatal(e)
	}
	if e = goose.Up(db, filepath.Join("..", "db", "migrations")); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { p.Close(); admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	return &core.Service{Pool: p}
}
func seed(t *testing.T, s *core.Service, account string) {
	t.Helper()
	for _, r := range core.Fixtures(account, time.Now().UTC()) {
		if e := s.Ingest(ctx, r); e != nil {
			t.Fatal(e)
		}
	}
}
func request(account string) api.Evaluation {
	return api.Evaluation{AccountId: api.EvaluationAccountId(account), OperationId: uuid.New(), Target: api.SinkA, Device: api.Known, Attestation: api.DemoValid}
}
func evaluate(t *testing.T, s *core.Service, r api.Evaluation) api.Decision {
	t.Helper()
	d, e := s.Evaluate(ctx, r)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func state(t *testing.T, s *core.Service, a string) api.State {
	t.Helper()
	v, e := s.State(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func command(t *testing.T, s *core.Service) string {
	t.Helper()
	cs, e := store.New(s.Pool).Pending(ctx)
	if e != nil || len(cs) != 1 {
		t.Fatalf("pending %+v %v", cs, e)
	}
	return cs[0].CommandID
}
func confirm(r api.Evaluation, d api.Decision) api.Confirmation {
	return api.Confirmation{AccountId: api.ConfirmationAccountId(r.AccountId), OperationId: r.OperationId, ConfirmationId: uuid.MustParse(*d.ConfirmationId), Proof: api.DemoApproved}
}

func TestC4HTTPScenarios(t *testing.T) {
	s := setup(t)
	risk := httptest.NewServer(transport.RiskRouter(s))
	defer risk.Close()
	gateway := httptest.NewServer(transport.GatewayRouter(risk.URL))
	defer gateway.Close()
	post := func(path string, b any, out any) int {
		t.Helper()
		payload, _ := json.Marshal(b)
		resp, e := http.Post(gateway.URL+path, "application/json", bytes.NewReader(payload))
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		if out != nil {
			if e = json.NewDecoder(resp.Body).Decode(out); e != nil {
				t.Fatal(e)
			}
		}
		return resp.StatusCode
	}
	for _, account := range []string{"demo-fraud", "demo-legal"} {
		seed(t, s, account)
		r := request(account)
		if account == "demo-fraud" {
			r.Device = api.New
		}
		var d api.Decision
		if status := post("/v1/evaluate", r, &d); status != 200 {
			t.Fatalf("status %d", status)
		}
		want := api.StepUp
		if account == "demo-fraud" {
			want = api.Hold
		}
		if d.Result != want {
			t.Fatalf("%+v", d)
		}
		if v := state(t, s, account); v.Target != "disabled" || v.Changes != 0 {
			t.Fatal(v)
		}
		if account == "demo-legal" {
			var allowed api.Decision
			if post("/v1/confirm", confirm(r, d), &allowed) != 200 || allowed.Result != api.Allow {
				t.Fatal(allowed)
			}
			if e := s.Drain(ctx); e != nil {
				t.Fatal(e)
			}
			if v := state(t, s, account); v.Target != "sink-a" || v.Changes != 1 {
				t.Fatal(v)
			}
		}
		v := state(t, s, account)
		if len(v.Decisions) == 0 || len(v.Audit) < 3 {
			t.Fatal(v)
		}
		resp, e := http.Get(gateway.URL + "/v1/state/" + account)
		if e != nil {
			t.Fatal(e)
		}
		var wire api.State
		if e = json.NewDecoder(resp.Body).Decode(&wire); e != nil {
			t.Fatal(e)
		}
		resp.Body.Close()
		if wire.Target != v.Target || len(wire.Audit) != len(v.Audit) {
			t.Fatal(wire)
		}
	}
	for _, body := range []any{map[string]any{}, map[string]any{"accountId": "real-number", "operationId": "bad"}, map[string]any{"accountId": "demo-clean", "operationId": uuid.NewString(), "target": "sink-a", "device": "known", "attestation": "demo-valid", "bypass": true}} {
		if status := post("/v1/evaluate", body, nil); status != 400 {
			t.Fatalf("invalid payload accepted: %d", status)
		}
	}
}
func TestStaleAuthorizationAndDuplicateDelivery(t *testing.T) {
	for _, restrict := range []bool{false, true} {
		t.Run(map[bool]string{false: "duplicate", true: "stale"}[restrict], func(t *testing.T) {
			s := setup(t)
			seed(t, s, "demo-clean")
			d := evaluate(t, s, request("demo-clean"))
			if d.Result != api.Allow {
				t.Fatal(d)
			}
			id := command(t, s)
			if restrict {
				if e := s.Hold(ctx, "demo-clean"); e != nil {
					t.Fatal(e)
				}
			}
			for i := 0; i < 2; i++ {
				if e := s.Deliver(ctx, id); e != nil {
					t.Fatal(e)
				}
			}
			want := 1
			if restrict {
				want = 0
			}
			v := state(t, s, "demo-clean")
			if v.Changes != want {
				t.Fatal(v)
			}
			count := 0
			for _, a := range v.Audit {
				if a["action"] == "execution" {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("execution audits %d", count)
			}
		})
	}
}
func TestConcurrentConfirmation(t *testing.T) {
	s := setup(t)
	seed(t, s, "demo-legal")
	r := request("demo-legal")
	d := evaluate(t, s, r)
	c := confirm(r, d)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, e := s.Confirm(ctx, c); results <- e }()
	}
	close(start)
	wg.Wait()
	close(results)
	ok, conflict := 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if errors.Is(e, core.ErrConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("success %d conflict %d", ok, conflict)
	}
	if e := s.Drain(ctx); e != nil {
		t.Fatal(e)
	}
	if v := state(t, s, "demo-legal"); v.Changes != 1 {
		t.Fatal(v)
	}
}
func TestBindingsAndExpiry(t *testing.T) {
	s := setup(t)
	seed(t, s, "demo-legal")
	r := request("demo-legal")
	d := evaluate(t, s, r)
	changed := r
	changed.Target = api.SinkB
	if _, e := s.Evaluate(ctx, changed); !errors.Is(e, core.ErrConflict) {
		t.Fatalf("changed params: %v", e)
	}
	c := confirm(r, d)
	c.AccountId = api.ConfirmationAccountIdDemoClean
	if _, e := s.Confirm(ctx, c); !errors.Is(e, core.ErrConflict) {
		t.Fatalf("wrong account %v", e)
	}
	c = confirm(r, d)
	c.OperationId = uuid.New()
	if _, e := s.Confirm(ctx, c); e == nil {
		t.Fatal("wrong operation accepted")
	}
	if _, e := s.Pool.Exec(ctx, "UPDATE operation_confirmation SET issued_at=clock_timestamp()-interval '3 minutes',expires_at=clock_timestamp()-interval '1 second'"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Confirm(ctx, confirm(r, d)); !errors.Is(e, core.ErrConflict) {
		t.Fatalf("expired accepted: %v", e)
	}
}
func TestLateSignalAndReplay(t *testing.T) {
	s := setup(t)
	seed(t, s, "demo-clean")
	r := request("demo-clean")
	old := evaluate(t, s, r)
	q := store.New(s.Pool)
	before, e := q.GetDecision(ctx, old.DecisionId)
	if e != nil {
		t.Fatal(e)
	}
	late := core.Fixtures("demo-clean", time.Now().UTC())[0]
	late.ID = uuid.NewString()
	late.OccurredAt = time.Now().Add(-40 * time.Minute)
	late.Cell = "late-cell"
	late.SIM = "sim-late"
	if e = s.Ingest(ctx, late); e != nil {
		t.Fatal(e)
	}
	if e = s.Ingest(ctx, late); e != nil {
		t.Fatal(e)
	}
	after, e := q.GetDecision(ctx, old.DecisionId)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(before.Evidence, after.Evidence) {
		t.Fatal("old evidence changed")
	}
	ds, e := q.Decisions(ctx, "demo-clean")
	if e != nil || len(ds) != 2 || ds[1].SupersedesDecisionID.String != old.DecisionId {
		t.Fatalf("%+v %v", ds, e)
	}
	late.SIM = "tampered"
	if e = s.Ingest(ctx, late); !errors.Is(e, core.ErrConflict) {
		t.Fatalf("replay hash %v", e)
	}
	if e = s.Drain(ctx); e != nil {
		t.Fatal(e)
	}
	if state(t, s, "demo-clean").Changes != 0 {
		t.Fatal("stale command executed")
	}
}
func TestAtomicAuditAndImmutableHistory(t *testing.T) {
	s := setup(t)
	seed(t, s, "demo-clean")
	r := request("demo-clean")
	// Force the mandatory audit write to fail after the decision insert.
	if _, e := s.Pool.Exec(ctx, "ALTER TABLE audit_event ADD CONSTRAINT reject_decision CHECK(action <> 'decision')"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Evaluate(ctx, r); e == nil {
		t.Fatal("audit failure accepted")
	}
	v := state(t, s, "demo-clean")
	if len(v.Decisions) != 0 || v.Changes != 0 {
		t.Fatal("partial decision committed")
	}
	if cs, e := store.New(s.Pool).Pending(ctx); e != nil || len(cs) != 0 {
		t.Fatal("outbox committed without audit")
	}
	if _, e := s.Pool.Exec(ctx, "ALTER TABLE audit_event DROP CONSTRAINT reject_decision"); e != nil {
		t.Fatal(e)
	}
	evaluate(t, s, r)
	for _, table := range []string{"risk_decision", "audit_event", "policy_version"} {
		if _, e := s.Pool.Exec(ctx, "DELETE FROM "+table); e == nil {
			t.Fatalf("%s history mutable", table)
		}
	}
}
func TestMigrationRoundtrip(t *testing.T) {
	s := setup(t)
	cfg := s.Pool.Config()
	db := stdlib.OpenDB(*cfg.ConnConfig)
	var e error
	defer db.Close()
	if e = goose.DownTo(db, "../db/migrations", 0); e != nil {
		t.Fatal(e)
	}
	if e = goose.Up(db, "../db/migrations"); e != nil {
		t.Fatal(e)
	}
}
func TestScorerUnavailable(t *testing.T) {
	g := httptest.NewServer(transport.GatewayRouter("http://127.0.0.1:1"))
	defer g.Close()
	b, _ := json.Marshal(request("demo-clean"))
	start := time.Now()
	r, e := http.Post(g.URL+"/v1/evaluate", "application/json", bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	if r.StatusCode != 503 || time.Since(start) > 2*time.Second {
		t.Fatalf("%d %s", r.StatusCode, time.Since(start))
	}
}

func TestRuntimeRoleACL(t *testing.T) {
	s := setup(t)
	cfg := s.Pool.Config()
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, e := c.Exec(ctx, "SET ROLE antifraud_runtime")
		return e
	}
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	runtime := &core.Service{Pool: p}
	seed(t, runtime, "demo-legal")
	r := request("demo-legal")
	d := evaluate(t, runtime, r)
	if _, e = runtime.Confirm(ctx, confirm(r, d)); e != nil {
		t.Fatal(e)
	}
	if e = runtime.Drain(ctx); e != nil {
		t.Fatal(e)
	}
	if state(t, runtime, "demo-legal").Changes != 1 {
		t.Fatal("runtime cannot complete workflow")
	}
	for _, query := range []string{"UPDATE risk_decision SET result='allow'", "DELETE FROM audit_event", "TRUNCATE risk_decision CASCADE", "UPDATE policy_version SET definition='{}'", "ALTER TABLE risk_decision DISABLE TRIGGER ALL", "DELETE FROM execution_result"} {
		if _, e = p.Exec(ctx, query); e == nil {
			t.Fatalf("runtime allowed %s", query)
		}
	}
}

func TestExpiredPermit(t *testing.T) {
	s := setup(t)
	seed(t, s, "demo-clean")
	r := request("demo-clean")
	q := store.New(s.Pool)
	// Construct a once-valid permit whose deadline has passed, without editing history.
	op := r.OperationId.String()
	b, _ := json.Marshal(r)
	ph := "synthetic-expired-hash"
	if _, e := q.SaveOperation(ctx, store.SaveOperationParams{OperationID: op, AccountID: "demo-clean", OperationType: core.OperationType, ParametersHash: ph, Request: b}); e != nil {
		t.Fatal(e)
	}
	g, e := q.LockGuard(ctx, "demo-clean")
	if e != nil {
		t.Fatal(e)
	}
	id := uuid.NewString()
	commandID := uuid.NewString()
	now := time.Now()
	if e = q.AddDecision(ctx, store.AddDecisionParams{DecisionID: id, AccountID: "demo-clean", OperationID: op, OperationType: core.OperationType, ParametersHash: ph, GuardRevision: g.Revision, PolicyVersion: "demo-v1", Result: "allow", DecidedAt: pgtype.Timestamptz{Time: now.Add(-3 * time.Minute), Valid: true}, ValidUntil: pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true}, Evidence: json.RawMessage(`{"schemaVersion":1}`)}); e != nil {
		t.Fatal(e)
	}
	if e = q.Enqueue(ctx, store.EnqueueParams{CommandID: commandID, OperationID: op, AccountID: "demo-clean", DecisionID: id}); e != nil {
		t.Fatal(e)
	}
	if e = s.Deliver(ctx, commandID); e != nil {
		t.Fatal(e)
	}
	v := state(t, s, "demo-clean")
	if v.Changes != 0 {
		t.Fatal("expired permit executed")
	}
	execution, e := q.GetExecution(ctx, commandID)
	if e != nil || execution.Result != "rejected" {
		t.Fatalf("%+v %v", execution, e)
	}
}
