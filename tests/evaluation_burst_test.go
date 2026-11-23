//go:build integration && evaluation

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	api "antifraud/internal/api/riskapi"
)

const burstRPS = 100
const burstDuration = 3 * time.Second
const burstWorkers = 40

type burstResult struct {
	OfferedRPS    int  `json:"offered_rps"`
	DurationMS    int  `json:"duration_ms"`
	Planned       int  `json:"planned"`
	Sent          int  `json:"sent"`
	Successful    int  `json:"successful"`
	OnTime        int  `json:"successful_in_window"`
	Dropped       int  `json:"dropped"`
	ForegroundFit bool `json:"foreground_in_window"`
}

func evaluationPost(client *http.Client, url string, value any, result *api.Decision) int {
	body, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	response, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK && json.NewDecoder(response.Body).Decode(result) != nil {
		return 0
	}
	return response.StatusCode
}

// This background account shares the gateway, scorer and PostgreSQL pool.
// Known fixtures plus a new device produce ordinary step-up checks without
// submitting confirmations or changing the foreground account's service.
func driveEvaluationBurst(client *http.Client, gatewayURL, account string, start time.Time) <-chan burstResult {
	done := make(chan burstResult, 1)
	go func() {
		defer close(done)
		end := start.Add(burstDuration)
		interval := time.Second / burstRPS
		planned := int(burstDuration / interval)
		result := burstResult{Planned: planned, OfferedRPS: burstRPS, DurationMS: int(burstDuration.Milliseconds())}
		workers := make(chan struct{}, burstWorkers)
		type completion struct{ success, onTime bool }
		observations := make(chan completion, planned)
		var pending sync.WaitGroup
		for slot := 0; slot < planned; slot++ {
			due := start.Add(time.Duration(slot) * interval)
			if delay := time.Until(due); delay > 0 {
				time.Sleep(delay)
			}
			if time.Since(due) >= interval {
				continue
			}
			select {
			case workers <- struct{}{}:
				result.Sent++
				pending.Add(1)
				go func() {
					defer pending.Done()
					defer func() { <-workers }()
					r := request(account)
					r.Device = api.New
					var decision api.Decision
					status := evaluationPost(client, gatewayURL+"/v1/evaluate", r, &decision)
					ok := status == http.StatusOK && decision.Result == api.StepUp &&
						decision.OperationId == r.OperationId.String()
					observations <- completion{ok, time.Now().Before(end)}
				}()
			default:
			}
		}
		pending.Wait()
		close(observations)
		for observation := range observations {
			if observation.success {
				result.Successful++
				if observation.onTime {
					result.OnTime++
				}
			}
		}
		result.Dropped = planned - result.Sent
		done <- result
	}()
	return done
}
