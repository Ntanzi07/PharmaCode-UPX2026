package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/config"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/router"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatal(err)
	}

	queries := db.New(pool)

	drugSvc := service.NewDrugService(queries)
	packageSvc := service.NewPackageService(queries)
	summarySvc := service.NewSummaryService(queries)

	drugH := handler.NewDrugHandler(drugSvc)
	packageH := handler.NewPackageHandler(packageSvc)
	summaryH := handler.NewSummaryHandler(summarySvc)

	r := router.New(drugH, packageH, summaryH)

	log.Printf("server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
