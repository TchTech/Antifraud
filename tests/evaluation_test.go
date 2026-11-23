//go:build integration && evaluation

package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	api "antifraud/internal/api/riskapi"
	"antifraud/internal/core"
	"antifraud/internal/transport"
)

// Evaluation cases use the existing synthetic C4 and legitimate-control fixtures.
// The initial decision and service state are observations, even when they differ
// from the expected outcome. Regression tests enforce hard invariants separately.
func TestPolicyEvaluation(t *testing.T) {
	cases := []struct {
		name, group, account string
		device               api.EvaluationDevice
		expected             api.DecisionResult
		legitimate           bool
		stationary           bool
	}{
		{"known-context", "ordinary", "demo-clean", api.Known, api.Allow, true, true},
		{"local-travel", "travel", "demo-clean", api.Known, api.Allow, true, false},
		{"legal-sim-replacement", "sim-replacement", "demo-legal", api.Known, api.StepUp, true, false},
		{"legal-new-device", "new-device", "demo-clean", api.New, api.StepUp, true, true},
		{"c4-protection", "c4", "demo-fraud", api.New, api.Hold, false, false},
	}
	anchor := time.Unix(0, 0).UTC()
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			s := setup(t)
			risk := httptest.NewServer(transport.RiskRouter(s))
			t.Cleanup(risk.Close)
			gateway := httptest.NewServer(transport.GatewayRouter(risk.URL))
			t.Cleanup(gateway.Close)
			fixtures := core.Fixtures(scenario.account, anchor)
			if len(fixtures) != 2 {
				t.Fatalf("evaluation requires two registrations, got %d", len(fixtures))
			}
			if scenario.stationary {
				fixtures[1].Cell = fixtures[0].Cell
				fixtures[1].Lat, fixtures[1].Lon = fixtures[0].Lat, fixtures[0].Lon
			}
			// Relative times and stable IDs identify the exact inputs across runs.
			// Ingest a copy with timestamps relative to the actual evaluation time.
			now := time.Now().UTC()
			for i, id := range [2]string{"evaluation-first", "evaluation-second"} {
				fixtures[i].ID = id
				registration := fixtures[i]
				registration.OccurredAt = now.Add(registration.OccurredAt.Sub(anchor))
				if err := s.Ingest(ctx, registration); err != nil {
					t.Fatalf("ingest synthetic registration: %v", err)
				}
			}

			r := request(scenario.account)
			r.Device = scenario.device
			client := &http.Client{Timeout: 3 * time.Second}
			t.Cleanup(client.CloseIdleConnections)
			var background <-chan burstResult
			var burstStart time.Time
			if os.Getenv("ANTIFRAUD_EVALUATION_BURST") == "1" {
				account := "demo-clean"
				if account == scenario.account {
					account = "demo-legal"
				}
				seed(t, s, account)
				burstStart = time.Now()
				background = driveEvaluationBurst(client, gateway.URL, account, burstStart)
				defer func() { <-background }()
				time.Sleep(time.Until(burstStart.Add(burstDuration / 2)))
			}
			foregroundStart := time.Now()
			var initial api.Decision
			status := evaluationPost(client, gateway.URL+"/v1/evaluate", r, &initial)
			final := initial
			problem := ""
			if status != http.StatusOK || !validEvaluationDecision(initial, r.OperationId.String()) {
				problem = "evaluate"
				initial, final = api.Decision{}, api.Decision{}
			} else if initial.Result == api.StepUp && scenario.legitimate {
				if initial.ConfirmationId == nil {
					problem = "confirmation_missing"
				} else {
					final = api.Decision{}
					status = evaluationPost(client, gateway.URL+"/v1/confirm", confirm(r, initial), &final)
					if status != http.StatusOK || !validEvaluationDecision(final, r.OperationId.String()) {
						problem = "confirm"
						final = api.Decision{}
					}
				}
			}
			if err := s.Drain(ctx); err != nil && problem == "" {
				problem = "outbox"
			}
			serviceState := evaluationState(client, gateway.URL+"/v1/state/"+scenario.account)
			if serviceState == nil && problem == "" {
				problem = "state"
			}
			foregroundEnd := time.Now()
			observation := map[string]any{
				"id": scenario.name, "group": scenario.group, "legitimate": scenario.legitimate,
				"expected": scenario.expected, "device": r.Device, "attestation": r.Attestation,
				"fixture": fixtures, "requested_target": r.Target,
				"initial": initial.Result, "final": final.Result, "policy": initial.PolicyVersion,
				"target": nil, "changes": nil, "problem": problem, "http_status": status,
				"foreground_ms": float64(foregroundEnd.Sub(foregroundStart).Microseconds()) / 1000,
			}
			if serviceState != nil {
				observation["target"], observation["changes"] = serviceState.Target, serviceState.Changes
			}
			if background != nil {
				load := <-background
				load.ForegroundFit = foregroundEnd.Before(burstStart.Add(burstDuration))
				observation["load"] = load
			}
			row, err := json.Marshal(observation)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("METRIC_CASE %s", row)
		})
	}
}

func validEvaluationDecision(d api.Decision, operation string) bool {
	return d.OperationId == operation && d.PolicyVersion != "" &&
		(d.Result == api.Allow || d.Result == api.StepUp || d.Result == api.Hold)
}

func evaluationState(client *http.Client, url string) *api.State {
	response, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	var actual api.State
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&actual) != nil || actual.Target == "" {
		return nil
	}
	return &actual
}
