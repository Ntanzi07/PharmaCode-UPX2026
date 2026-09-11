package main

import (
	"PharmaCode_UPX2026/internal/config"
	"PharmaCode_UPX2026/internal/db"
	"PharmaCode_UPX2026/internal/handler"
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)

	queries := db.New(pool)

	h := handler.NewDrugHandler(queries)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /drugs/ean/{ean}", h.GetByEAN)

	log.Printf("server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}
