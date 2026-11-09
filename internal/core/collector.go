package core

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	api "antifraud/internal/api/riskapi"
	"antifraud/internal/policy"
	"antifraud/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Registration struct {
	ID            string    `json:"id"`
	Account       string    `json:"account"`
	Source        string    `json:"source"`
	SchemaVersion int32     `json:"schemaVersion"`
	SIM           string    `json:"sim"`
	Cell          string    `json:"cell"`
	Lat           float64   `json:"lat"`
	Lon           float64   `json:"lon"`
	OccurredAt    time.Time `json:"occurredAt"`
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	a := math.Pow(math.Sin((lat2-lat1)*rad/2), 2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Pow(math.Sin((lon2-lon1)*rad/2), 2)
	return 6371 * 2 * math.Asin(math.Sqrt(math.Min(1, a)))
}
func (s *Service) Ingest(ctx context.Context, r Registration) error {
	if r.Source != "synthetic-network" || r.SchemaVersion != 1 || r.ID == "" || r.SIM == "" || r.Cell == "" || r.OccurredAt.IsZero() || r.OccurredAt.After(time.Now().Add(time.Minute)) || math.Abs(r.Lat) > 90 || math.Abs(r.Lon) > 180 {
		return ErrConflict
	}
	return s.transaction(ctx, r.Account, func(q *store.Queries, g store.AccountGuard) error {
		ph := hash(r)
		n, e := q.Receipt(ctx, store.ReceiptParams{Source: r.Source, EventID: r.ID, PayloadHash: ph})
		if e != nil {
			return e
		}
		if n == 0 {
			h, e := q.ReceiptHash(ctx, store.ReceiptHashParams{Source: r.Source, EventID: r.ID})
			if e != nil {
				return e
			}
			if h != ph {
				return ErrConflict
			}
			return nil
		}
		n, e = q.AddRegistration(ctx, store.AddRegistrationParams{EventID: r.ID, AccountID: r.Account, Source: r.Source, SchemaVersion: r.SchemaVersion, SimID: r.SIM, CellID: r.Cell, Lat: r.Lat, Lon: r.Lon, EventTime: ts(r.OccurredAt)})
		if e != nil {
			return e
		}
		if n == 0 {
			return nil
		}
		rows, e := q.RecentRegistrations(ctx, r.Account)
		if e != nil {
			return e
		}
		last := rows[0]
		p := policy.Profile{Source: last.Source, EventID: last.EventID, OccurredAt: last.EventTime.Time, ReceivedAt: last.ReceivedAt.Time}
		if len(rows) > 1 {
			prev := rows[1]
			changed := prev.SimID != last.SimID
			hours := last.EventTime.Time.Sub(prev.EventTime.Time).Hours()
			km := distance(prev.Lat, prev.Lon, last.Lat, last.Lon)
			speed := 0.0
			if hours > 0 {
				speed = km / hours
			} else if km > 0 {
				speed = 1e9
			}
			teleport := speed > policy.TeleportKmh
			p.SIMChanged = &changed
			p.Teleportation = &teleport
			p.SpeedKmh = &speed
		}
		if e = q.SaveProfile(ctx, store.SaveProfileParams{AccountID: r.Account, Snapshot: raw(p)}); e != nil {
			return e
		}
		// Any newly received signal invalidates previously issued permits.
		g, e = q.ChangeGuard(ctx, store.ChangeGuardParams{AccountID: r.Account, Held: g.Held})
		if e != nil {
			return e
		}
		if e = audit(ctx, q, r.Account, "", "signal", map[string]any{"event": r, "profile": p}); e != nil {
			return e
		}
		prior, e := q.LatestDecision(ctx, r.Account)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		op, e := q.GetOperation(ctx, prior.OperationID)
		if e != nil {
			return e
		}
		var request api.Evaluation
		if e = json.Unmarshal(op.Request, &request); e != nil {
			return e
		}
		_, e = s.evaluate(ctx, q, g, request, prior.DecisionID)
		return e
	})
}

// Fixtures model C4 travel distances and relative times; all identifiers are synthetic.
func Fixtures(account string, now time.Time) []Registration {
	lat, lon := 54.7104, 20.4522
	first := Registration{ID: uuid.NewString(), Account: account, Source: "synthetic-network", SchemaVersion: 1, SIM: "sim-old-" + account, Cell: "cell-kaliningrad", Lat: lat, Lon: lon, OccurredAt: now.Add(-85 * time.Minute)}
	second := first
	second.ID = uuid.NewString()
	second.Cell = "cell-svetlogorsk"
	second.Lat = 54.943
	second.Lon = 20.15
	second.OccurredAt = now.Add(-25 * time.Minute)
	if account != "demo-clean" {
		second.SIM = "sim-new-" + account
	}
	if account == "demo-fraud" {
		second.Lat = 43.1155
		second.Lon = 131.8855
		second.Cell = "cell-vladivostok"
		second.OccurredAt = first.OccurredAt.Add(25 * time.Minute)
	}
	return []Registration{first, second}
}
