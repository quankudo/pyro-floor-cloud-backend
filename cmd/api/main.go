package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/pyrofloor/cloud-backend/internal/config"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/auth"
	"github.com/pyrofloor/cloud-backend/internal/module/canhbao"
	canhbaopostgres "github.com/pyrofloor/cloud-backend/internal/module/canhbao/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/khuvuc"
	khuvucpostgres "github.com/pyrofloor/cloud-backend/internal/module/khuvuc/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/loaikhuvuc"
	loaikhuvucpostgres "github.com/pyrofloor/cloud-backend/internal/module/loaikhuvuc/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/loaimohinh"
	loaimohinhpostgres "github.com/pyrofloor/cloud-backend/internal/module/loaimohinh/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/nguoidung"
	nguoidungpostgres "github.com/pyrofloor/cloud-backend/internal/module/nguoidung/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/phanconggiamsat"
	phanconggiamsatpostgres "github.com/pyrofloor/cloud-backend/internal/module/phanconggiamsat/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/sos"
	sospostgres "github.com/pyrofloor/cloud-backend/internal/module/sos/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/tang"
	tangpostgres "github.com/pyrofloor/cloud-backend/internal/module/tang/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/thanhvienkhuvuc"
	thanhvienkhuvucpostgres "github.com/pyrofloor/cloud-backend/internal/module/thanhvienkhuvuc/postgres"
	"github.com/pyrofloor/cloud-backend/internal/module/toanha"
	toanhapostgres "github.com/pyrofloor/cloud-backend/internal/module/toanha/postgres"
	"github.com/pyrofloor/cloud-backend/internal/router"
)

func main() {
	// Load .env for local development.
	// In production, environment variables may be injected externally.
	if err := godotenv.Load(".env.dev"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	pool, err := database.New(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer pool.Close()

	tokenManager, err := auth.NewTokenManager(auth.TokenConfig{
		AccessSecret:  cfg.Auth.AccessTokenSecret,
		RefreshSecret: cfg.Auth.RefreshTokenSecret,
		Issuer:        "cloud-backend",
		Audience:      "cloud-backend-api",
		AccessTTL:     cfg.Auth.AccessTokenTTL,
		RefreshTTL:    cfg.Auth.RefreshTokenTTL,
		MaxSessionTTL: 30 * 24 * time.Hour,
		ClockSkew:     30 * time.Second,
	})
	if err != nil {
		log.Fatalf("configure authentication tokens: %v", err)
	}

	authMiddleware, err := middleware.NewAuth(tokenManager)
	if err != nil {
		log.Fatalf("configure authentication middleware: %v", err)
	}

	userRepository := nguoidungpostgres.NewNguoiDungRepository(pool)
	authService := auth.NewService(userRepository, tokenManager)
	authHandler, err := auth.NewAuthHandler(authService, tokenManager)
	if err != nil {
		log.Fatalf("configure authentication handler: %v", err)
	}
	membershipRepository := thanhvienkhuvucpostgres.NewRepository()
	membershipService, err := thanhvienkhuvuc.NewService(membershipRepository)
	if err != nil {
		log.Fatalf("configure area membership service: %v", err)
	}
	supervisorAssignmentRepository := phanconggiamsatpostgres.NewRepository(pool)
	supervisorAssignmentService, err := phanconggiamsat.NewService(supervisorAssignmentRepository)
	if err != nil {
		log.Fatalf("configure supervisor assignment service: %v", err)
	}
	supervisorAssignmentHandler, err := phanconggiamsat.NewHandler(supervisorAssignmentService)
	if err != nil {
		log.Fatalf("configure supervisor assignment handler: %v", err)
	}
	toaNhaRepository := toanhapostgres.NewToaNhaNguoiDungRepository(pool)
	toaNhaService := toanha.NewToaNhaService(toaNhaRepository)
	toaNhaHandler, err := toanha.NewToaNhaHandler(toaNhaService)
	if err != nil {
		log.Fatalf("configure toa nha handler: %v", err)
	}

	floorRepository, err := tangpostgres.NewRepository(pool)
	if err != nil {
		log.Fatalf("configure floor repository: %v", err)
	}
	floorService, err := tang.NewService(floorRepository)
	if err != nil {
		log.Fatalf("configure floor service: %v", err)
	}
	floorHandler, err := tang.NewHandler(floorService)
	if err != nil {
		log.Fatalf("configure floor handler: %v", err)
	}

	batchUserService, err := nguoidung.NewService(
		userRepository,
		membershipService,
		supervisorAssignmentService,
		toaNhaService,
		auth.HashPassword,
		userRepository,
		floorService,
	)
	if err != nil {
		log.Fatalf("configure batch user service: %v", err)
	}
	userHandler, err := nguoidung.NewNguoiDungHandler(batchUserService, batchUserService)
	if err != nil {
		log.Fatalf("configure user handler: %v", err)
	}

	areaRepository, err := khuvucpostgres.NewRepository(pool)
	if err != nil {
		log.Fatalf("configure area repository: %v", err)
	}
	areaService, err := khuvuc.NewService(areaRepository)
	if err != nil {
		log.Fatalf("configure area service: %v", err)
	}
	areaHandler, err := khuvuc.NewHandler(areaService)
	if err != nil {
		log.Fatalf("configure area handler: %v", err)
	}

	alertRepository, err := canhbaopostgres.NewRepository(pool)
	if err != nil {
		log.Fatalf("configure alert repository: %v", err)
	}
	alertService, err := canhbao.NewService(alertRepository)
	if err != nil {
		log.Fatalf("configure alert service: %v", err)
	}
	alertHandler, err := canhbao.NewHandler(alertService)
	if err != nil {
		log.Fatalf("configure alert handler: %v", err)
	}

	sosRepository, err := sospostgres.NewRepository(pool)
	if err != nil {
		log.Fatalf("configure SOS repository: %v", err)
	}
	sosService, err := sos.NewService(sosRepository)
	if err != nil {
		log.Fatalf("configure SOS service: %v", err)
	}
	sosHandler, err := sos.NewHandler(sosService)
	if err != nil {
		log.Fatalf("configure SOS handler: %v", err)
	}

	modelTypeRepository, err := loaimohinhpostgres.NewRepository(pool)
	if err != nil {
		log.Fatalf("configure model type repository: %v", err)
	}
	modelTypeService, err := loaimohinh.NewService(modelTypeRepository)
	if err != nil {
		log.Fatalf("configure model type service: %v", err)
	}
	modelTypeHandler, err := loaimohinh.NewHandler(modelTypeService)
	if err != nil {
		log.Fatalf("configure model type handler: %v", err)
	}

	areaTypeRepository, err := loaikhuvucpostgres.NewRepository(pool)
	if err != nil {
		log.Fatalf("configure area type repository: %v", err)
	}
	areaTypeService, err := loaikhuvuc.NewService(areaTypeRepository)
	if err != nil {
		log.Fatalf("configure area type service: %v", err)
	}
	areaTypeHandler, err := loaikhuvuc.NewHandler(areaTypeService)
	if err != nil {
		log.Fatalf("configure area type handler: %v", err)
	}

	apiRouter, err := router.New(router.Config{
		Health:         pool,
		Auth:           authHandler,
		NguoiDung:      userHandler,
		Assignments:    supervisorAssignmentHandler,
		Alerts:         alertHandler,
		SOS:            sosHandler,
		Areas:          areaHandler,
		Floors:         floorHandler,
		LoaiKhuVuc:     areaTypeHandler,
		LoaiMoHinh:     modelTypeHandler,
		ToaNha:         toaNhaHandler,
		Authentication: authMiddleware,
	})
	if err != nil {
		log.Fatalf("configure HTTP router: %v", err)
	}

	cors, err := middleware.NewCORS(middleware.CORSConfig{
		AllowedOrigins:                cfg.CORSAllowedOrigins,
		RequireOriginForUnsafeMethods: false,
	})
	if err != nil {
		log.Fatalf("configure CORS: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           cors.Middleware(apiRouter),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("API starting on port %s", cfg.AppPort)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	case <-ctx.Done():
		log.Println("Shutting down API...")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = server.Close()
	}

	log.Println("API stopped")
}
