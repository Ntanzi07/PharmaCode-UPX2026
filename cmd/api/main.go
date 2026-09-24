package main

import (
	"context"
	"log"
	"net/http"
	"time"

	_ "github.com/Ntanzi07/PharmaCode-UPX2026/docs"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/config"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/router"
	"github.com/jackc/pgx/v5/pgxpool"
)

// @title           PharmaCode API
// @version         1.0
// @description     API that reads the barcode on a medicine box and returns the simplified leaflet.
// @description     Public route: GET /drugs/ean/{ean}. All others require login (POST /auth/login sets a session cookie).
// @BasePath        /
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
	userSvc := service.NewUserService(queries)
	importSvc := service.NewImportService(pool)
	authSvc := auth.NewService(queries, cfg.SessionTTL)

	bootstrapAdmin(userSvc, cfg)

	// 5 wrong passwords within 15 minutes lock that IP + email
	limiter := auth.NewLoginLimiter(5, 15*time.Minute)

	r := router.New(router.Handlers{
		Drug:    handler.NewDrugHandler(drugSvc),
		Package: handler.NewPackageHandler(packageSvc),
		Summary: handler.NewSummaryHandler(summarySvc),
		Auth:    handler.NewAuthHandler(authSvc, userSvc, limiter, cfg.CookieSecure),
		User:    handler.NewUserHandler(userSvc),
		Import:  handler.NewImportHandler(importSvc),
	}, auth.NewMiddleware(authSvc))

	log.Printf("server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}

// bootstrapAdmin creates the first admin from ADMIN_EMAIL / ADMIN_PASSWORD
// when the database has no users yet. After that, the variables are ignored.
func bootstrapAdmin(users *service.UserService, cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		if n, err := users.List(ctx); err == nil && len(n) == 0 {
			log.Printf("WARNING: no users yet; set ADMIN_EMAIL and ADMIN_PASSWORD to create the first admin")
		}
		return
	}
	created, err := users.EnsureAdmin(ctx, cfg.AdminName, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		log.Fatalf("failed to create first admin: %v", err)
	}
	if created {
		log.Printf("first admin created: %s", cfg.AdminEmail)
	}
}
