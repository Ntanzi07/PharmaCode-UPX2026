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
// @description     API que lê o código da caixa do remédio e retorna a bula simplificada.
// @description     Rota pública: GET /drugs/ean/{ean}. As demais exigem login (POST /auth/login grava um cookie de sessão).
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
	authSvc := auth.NewService(queries, cfg.SessionTTL)

	bootstrapAdmin(userSvc, cfg)

	// 5 senhas erradas em 15 minutos bloqueiam aquele IP + email
	limiter := auth.NewLoginLimiter(5, 15*time.Minute)

	r := router.New(router.Handlers{
		Drug:    handler.NewDrugHandler(drugSvc),
		Package: handler.NewPackageHandler(packageSvc),
		Summary: handler.NewSummaryHandler(summarySvc),
		Auth:    handler.NewAuthHandler(authSvc, userSvc, limiter, cfg.CookieSecure),
		User:    handler.NewUserHandler(userSvc),
	}, auth.NewMiddleware(authSvc))

	log.Printf("server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}

// bootstrapAdmin cria o primeiro admin a partir de ADMIN_EMAIL / ADMIN_PASSWORD
// quando o banco ainda não tem nenhum usuário. Depois disso, as variáveis são ignoradas.
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
