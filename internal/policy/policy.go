// Package policy contains reversible demo rules, independent of HTTP and storage.
package policy

import "time"

const Version = "demo-v1"
const Freshness = 24 * time.Hour
const TeleportKmh = 1000.0

type Profile struct {
	Source        string    `json:"source"`
	EventID       string    `json:"eventId"`
	OccurredAt    time.Time `json:"occurredAt"`
	ReceivedAt    time.Time `json:"receivedAt"`
	SIMChanged    *bool     `json:"simChanged"`
	Teleportation *bool     `json:"teleportation"`
	SpeedKmh      *float64  `json:"speedKmh"`
}

type Features struct {
	SchemaVersion int     `json:"schemaVersion"`
	Profile       Profile `json:"network"`
	Device        string  `json:"device"`
	Attestation   string  `json:"attestation"`
	Fresh         bool    `json:"fresh"`
	Confirmation  string  `json:"confirmation"`
	Reason        string  `json:"reason"`
}

func Calculate(p Profile, device, attestation string, now time.Time) Features {
	return Features{SchemaVersion: 1, Profile: p, Device: device, Attestation: attestation, Fresh: !p.OccurredAt.IsZero() && !p.OccurredAt.After(now) && now.Sub(p.OccurredAt) <= Freshness}
}

func Decide(f Features) (string, string) {
	if !f.Fresh || f.Profile.SIMChanged == nil || f.Profile.Teleportation == nil || f.Attestation != "demo-valid" {
		return "hold", "unknown_or_stale_signal"
	}
	if *f.Profile.SIMChanged && *f.Profile.Teleportation && f.Device == "new" {
		return "hold", "sim_teleport_new_device"
	}
	if *f.Profile.SIMChanged || f.Device == "new" || *f.Profile.Teleportation {
		return "step_up", "independent_confirmation_required"
	}
	return "allow", "known_context"
}
