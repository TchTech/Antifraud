package main

import (
	"antifraud/internal/core"
	"antifraud/internal/store"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"github.com/jackc/pgx/v5"
	"log"
	"os"
	"time"
)

func main() {
	file := flag.String("file", "", "JSON array of synthetic registrations")
	once := flag.Bool("once", false, "seed fixtures then exit")
	flag.Parse()
	ctx := context.Background()
	p, e := core.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		log.Fatal(e)
	}
	defer p.Close()
	s := &core.Service{Pool: p}
	var events []core.Registration
	if *file != "" {
		b, e := os.ReadFile(*file)
		if e != nil {
			log.Fatal(e)
		}
		if e = json.Unmarshal(b, &events); e != nil {
			log.Fatal(e)
		}
	} else {
		for _, a := range []string{"demo-fraud", "demo-legal", "demo-clean"} {
			if _, err := store.New(p).GetProfile(ctx, a); err == nil {
				continue
			} else if !errors.Is(err, pgx.ErrNoRows) {
				log.Fatal(err)
			}
			events = append(events, core.Fixtures(a, time.Now().UTC())...)
		}
	}
	for _, r := range events {
		if e = s.Ingest(ctx, r); e != nil {
			log.Fatal(e)
		}
	}
	log.Printf("collected %d synthetic registrations", len(events))
	if *once || *file != "" {
		return
	}
	// Fixed source seeds once. No operator network is contacted.
	select {}
}
