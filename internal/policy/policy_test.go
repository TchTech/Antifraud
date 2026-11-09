package policy

import (
	"testing"
	"time"
)

func TestPolicy(t *testing.T) {
	yes, no := true, false
	now := time.Now()
	for _, tc := range []struct {
		name                      string
		p                         Profile
		device, attestation, want string
	}{
		{"missing", Profile{}, "known", "demo-valid", "hold"},
		{"negative", Profile{OccurredAt: now, SIMChanged: &no, Teleportation: &no}, "known", "demo-valid", "allow"},
		{"legal", Profile{OccurredAt: now, SIMChanged: &yes, Teleportation: &no}, "known", "demo-valid", "step_up"},
		{"fraud", Profile{OccurredAt: now, SIMChanged: &yes, Teleportation: &yes}, "new", "demo-valid", "hold"},
		{"single geography", Profile{OccurredAt: now, SIMChanged: &no, Teleportation: &yes}, "known", "demo-valid", "step_up"},
		{"stale", Profile{OccurredAt: now.Add(-25 * time.Hour), SIMChanged: &no, Teleportation: &no}, "known", "demo-valid", "hold"},
		{"unknown attestation", Profile{OccurredAt: now, SIMChanged: &no, Teleportation: &no}, "known", "unknown", "hold"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := Decide(Calculate(tc.p, tc.device, tc.attestation, now))
			if got != tc.want {
				t.Fatalf("%s != %s", got, tc.want)
			}
		})
	}
}
