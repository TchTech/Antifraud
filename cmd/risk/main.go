package main

import (
	"antifraud/internal/core"
	"antifraud/internal/transport"
	"context"
	"database/sql"
	"flag"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	migration := flag.String("migrate", "", "up, down or status; uses db/migrations")
	flag.Parse()
	url := os.Getenv("DATABASE_URL")
	if *migration != "" {
		db, e := sql.Open("pgx", url)
		if e != nil {
			log.Fatal(e)
		}
		defer db.Close()
		if e = goose.SetDialect("postgres"); e != nil {
			log.Fatal(e)
		}
		if e = goose.Run(*migration, db, "db/migrations"); e != nil {
			log.Fatal(e)
		}
		return
	}
	ctx := context.Background()
	pool, e := core.Open(ctx, url)
	if e != nil {
		log.Fatal(e)
	}
	defer pool.Close()
	s := &core.Service{Pool: pool}
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if e := s.Drain(ctx); e != nil {
				log.Printf("outbox retry: %v", e)
			}
			cancel()
		}
	}()
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}
	server := &http.Server{Addr: addr, Handler: transport.RiskRouter(s), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
