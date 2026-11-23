//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "antifraud/internal/api/riskapi"
	"antifraud/internal/core"
	"antifraud/internal/policy"
	"antifraud/internal/store"
	"antifraud/internal/transport"
)

// TestSyntheticProtectionControls exercises real gateway decisions and real
// simulator state using only legitimate operations and controlled signal inputs.
func TestSyntheticProtectionControls(t *testing.T) {
	controls := []struct {
		name          string
		account       string
		device        api.EvaluationDevice
		registrations int
		signalAge     time.Duration
		want          api.DecisionResult
		wantReason    string
		wantFresh     bool
	}{
		{"known-context", "demo-clean", api.Known, 2, 0, api.Allow, "known_context", true},
		{"legitimate-sim-replacement", "demo-legal", api.Known, 2, 0, api.StepUp, "independent_confirmation_required", true},
		{"legitimate-new-device", "demo-clean", api.New, 2, 0, api.StepUp, "independent_confirmation_required", true},
		{"network-unavailable", "demo-clean", api.Known, 0, 0, api.Hold, "unknown_or_stale_signal", false},
		{"network-profile-incomplete", "demo-clean", api.Known, 1, 0, api.Hold, "unknown_or_stale_signal", true},
		{"network-profile-stale", "demo-clean", api.Known, 2, 48 * time.Hour, api.Hold, "unknown_or_stale_signal", false},
	}
	for _, control := range controls {
		t.Run(control.name, func(t *testing.T) {
			s := setup(t)
			q := store.New(s.Pool)
			risk := httptest.NewServer(transport.RiskRouter(s))
			t.Cleanup(risk.Close)
			gateway := transport.GatewayRouter(risk.URL)
			registrations := core.Fixtures(control.account, time.Now().UTC().Add(-control.signalAge))[:control.registrations]
			for _, registration := range registrations {
				if err := s.Ingest(ctx, registration); err != nil {
					t.Fatalf("ingest synthetic registration: %v", err)
				}
			}

			r := request(control.account)
			r.Device = control.device
			decision := postControlDecision(t, gateway, "/v1/evaluate", r)
			if decision.Result != control.want || decision.OperationId != r.OperationId.String() || decision.PolicyVersion != policy.Version {
				t.Fatalf("decision %+v, want %s for operation %s under %s", decision, control.want, r.OperationId, policy.Version)
			}
			assertControlEvidence(t, q, decision, registrations, control.wantReason, control.wantFresh)
			assertControlState(t, gateway, control.account, "disabled", 0, "", "")

			if control.want == api.StepUp {
				if decision.ConfirmationId == nil {
					t.Fatal("step-up did not issue a confirmation")
				}
				if err := s.Drain(ctx); err != nil {
					t.Fatalf("drain before confirmation: %v", err)
				}
				assertControlState(t, gateway, control.account, "disabled", 0, "", "")
				decision = postControlDecision(t, gateway, "/v1/confirm", confirm(r, decision))
				if decision.Result != api.Allow || decision.OperationId != r.OperationId.String() || decision.PolicyVersion != policy.Version {
					t.Fatalf("confirmed decision %+v, want allow for operation %s under %s", decision, r.OperationId, policy.Version)
				}
			} else if decision.ConfirmationId != nil {
				t.Fatalf("unexpected confirmation for %s", decision.Result)
			}

			pending, err := q.Pending(ctx)
			if err != nil {
				t.Fatalf("read outbox: %v", err)
			}
			wantTarget, wantChanges, commandID := "disabled", 0, ""
			if decision.Result == api.Allow {
				if len(pending) != 1 || pending[0].DecisionID != decision.DecisionId || pending[0].OperationID != r.OperationId.String() {
					t.Fatalf("outbox %+v, want one command for decision %s and operation %s", pending, decision.DecisionId, r.OperationId)
				}
				commandID = pending[0].CommandID
				wantTarget, wantChanges = string(r.Target), 1
			} else if len(pending) != 0 {
				t.Fatalf("outbox has %d commands for %s", len(pending), decision.Result)
			}
			if err := s.Drain(ctx); err != nil {
				t.Fatalf("drain outbox after final decision: %v", err)
			}
			assertControlState(t, gateway, control.account, wantTarget, wantChanges, commandID, decision.DecisionId)
			t.Logf("decision=%s final_decision=%s target=%s changes=%d", control.want, decision.Result, wantTarget, wantChanges)
		})
	}
}

func postControlDecision(t *testing.T, gateway http.Handler, path string, input any) api.Decision {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("encode decision request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s returned %d: %s", path, response.Code, response.Body)
	}
	var decision api.Decision
	if err := json.Unmarshal(response.Body.Bytes(), &decision); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	return decision
}

func assertControlEvidence(t *testing.T, q *store.Queries, decision api.Decision, registrations []core.Registration, reason string, fresh bool) {
	t.Helper()
	stored, err := q.GetDecision(ctx, decision.DecisionId)
	if err != nil {
		t.Fatalf("read persisted decision: %v", err)
	}
	if stored.PolicyVersion != decision.PolicyVersion || stored.Result != string(decision.Result) {
		t.Fatalf("persisted decision %s/%s, gateway returned %s/%s", stored.PolicyVersion, stored.Result, decision.PolicyVersion, decision.Result)
	}
	var evidence policy.Features
	if err := json.Unmarshal(stored.Evidence, &evidence); err != nil {
		t.Fatalf("decode persisted evidence: %v", err)
	}
	if evidence.SchemaVersion != 1 || evidence.Reason != reason || evidence.Fresh != fresh {
		t.Fatalf("evidence %+v, want schema 1, reason %s and fresh=%t", evidence, reason, fresh)
	}
	profile := evidence.Profile
	if len(registrations) > 0 {
		last := registrations[len(registrations)-1]
		if profile.Source != last.Source || profile.EventID != last.ID || profile.ReceivedAt.Before(profile.OccurredAt) || profile.OccurredAt.Sub(last.OccurredAt).Abs() > time.Microsecond {
			t.Fatalf("profile evidence does not describe latest synthetic registration: %+v", profile)
		}
	}
	if len(registrations) < 2 {
		if profile.SIMChanged != nil || profile.Teleportation != nil {
			t.Fatalf("unknown signals stored as known: %+v", profile)
		}
		return
	}
	changed := registrations[0].SIM != registrations[1].SIM
	if profile.SIMChanged == nil || *profile.SIMChanged != changed || profile.Teleportation == nil || *profile.Teleportation {
		t.Fatalf("expected known simChanged=%t and teleportation=false: %+v", changed, profile)
	}
}

func assertControlState(t *testing.T, gateway http.Handler, account, target string, changes int, commandID, decisionID string) {
	t.Helper()
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/state/"+account, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("state returned %d: %s", response.Code, response.Body)
	}
	var state api.State
	if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if state.Target != target || state.Changes != changes {
		t.Fatalf("service target=%s changes=%d, want target=%s changes=%d", state.Target, state.Changes, target, changes)
	}
	executions := 0
	for _, entry := range state.Audit {
		if entry["action"] != "execution" {
			continue
		}
		executions++
		details, ok := entry["details"].(map[string]any)
		if !ok || commandID == "" || details["commandId"] != commandID || details["decisionId"] != decisionID || details["result"] != "executed" {
			t.Fatalf("unexpected execution audit: %+v", entry)
		}
	}
	if executions != changes {
		t.Fatalf("got %d execution audit entries, want %d", executions, changes)
	}
}
