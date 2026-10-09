package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"docproject/backend/db"
	"docproject/backend/internal/accessperiods"
	"docproject/backend/internal/auth"
	"docproject/backend/internal/branchadmins"
	"docproject/backend/internal/config"
	"docproject/backend/internal/documents"
	"docproject/backend/internal/groups"
	"docproject/backend/internal/subgroupassignments"
	"docproject/backend/internal/subgroups"
	"docproject/backend/internal/users"
)

func main() {
	if err := run(); err != nil {
		log.Printf("application stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	)

	database, err := db.New(cfg)
	if err != nil {
		return err
	}
	defer database.Close()

	mux := http.NewServeMux()

	authService := auth.NewService(database)

	tokenService := auth.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTAccessTokenTTL,
	)

	authHandler := auth.NewHandler(
		authService,
		tokenService,
		logger,
	)

	// Authentication routes
	mux.Handle(
		"POST /auth/login",
		authHandler,
	)

	mux.Handle(
		"GET /auth/me",
		auth.AuthMiddleware(
			tokenService,
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					claims, ok := auth.ClaimsFromContext(
						r.Context(),
					)
					if !ok {
						writeJSON(
							w,
							http.StatusUnauthorized,
							map[string]string{
								"error": "unauthorized",
							},
						)
						return
					}

					writeJSON(
						w,
						http.StatusOK,
						map[string]any{
							"user_id":    claims.UserID,
							"college_id": claims.CollegeID,
							"role":       claims.Role,
						},
					)
				},
			),
		),
	)

	// Group routes
	groupsService := groups.NewService(database)
	groupsHandler := groups.NewHandler(groupsService)

	protectedGroupsHandler := auth.AuthMiddleware(
		tokenService,
		groupsHandler,
	)

	mux.Handle(
		"/groups",
		protectedGroupsHandler,
	)

	mux.Handle(
		"/groups/",
		protectedGroupsHandler,
	)

	// User routes
	usersService := users.NewService(database)
	usersHandler := users.NewHandler(usersService)

	mux.Handle(
		"/users",
		auth.AuthMiddleware(
			tokenService,
			usersHandler,
		),
	)

	// Branch admin routes
	branchAdminsService := branchadmins.NewService(database)

	branchAdminsHandler := branchadmins.NewHandler(
		branchAdminsService,
	)

	protectedBranchAdminsHandler := auth.AuthMiddleware(
		tokenService,
		branchAdminsHandler,
	)

	mux.Handle(
		"POST /branch-admins",
		protectedBranchAdminsHandler,
	)

	// Subgroup routes
	subgroupsService := subgroups.NewService(database)
	subgroupsHandler := subgroups.NewHandler(
		subgroupsService,
	)

	protectedSubgroupsHandler := auth.AuthMiddleware(
		tokenService,
		subgroupsHandler,
	)

	mux.Handle(
		"POST /groups/{group_id}/subgroups",
		protectedSubgroupsHandler,
	)

	mux.Handle(
		"GET /groups/{group_id}/subgroups",
		protectedSubgroupsHandler,
	)

	// Access-period routes
	accessPeriodsService := accessperiods.NewService(
		database,
	)

	accessPeriodsHandler := accessperiods.NewHandler(
		accessPeriodsService,
	)

	protectedAccessPeriodsHandler := auth.AuthMiddleware(
		tokenService,
		accessPeriodsHandler,
	)

	mux.Handle(
		"POST /groups/{group_id}/subgroups/{subgroup_id}/access-periods",
		protectedAccessPeriodsHandler,
	)

	mux.Handle(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/access-periods",
		protectedAccessPeriodsHandler,
	)

	// Document storage
	documentsStorage, err := documents.NewFileStorage(
		"storage/documents",
	)
	if err != nil {
		return err
	}

	// Document service
	documentsService := documents.NewServiceWithStorageAndAuthorizer(
		database,
		documentsStorage,
		accessPeriodsService,
	)

	// Document handler
	documentsHandler := documents.NewHandler(
		documentsService,
	)

	// Document routes
	mux.Handle(
		"/documents",
		auth.AuthMiddleware(
			tokenService,
			documentsHandler,
		),
	)

	mux.Handle(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/documents",
		auth.AuthMiddleware(
			tokenService,
			documentsHandler,
		),
	)

	mux.Handle(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/documents/{document_id}/preview",
		auth.AuthMiddleware(
			tokenService,
			documentsHandler,
		),
	)

	mux.Handle(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/documents/{document_id}/download",
		auth.AuthMiddleware(tokenService, documentsHandler),
	)

	mux.Handle(
		"/documents/",
		auth.AuthMiddleware(
			tokenService,
			documentsHandler,
		),
	)

	// Subgroup faculty assignment routes
	subgroupAssignmentsService := subgroupassignments.NewService(
		database,
	)

	subgroupAssignmentsHandler := subgroupassignments.NewHandler(
		subgroupAssignmentsService,
	)

	protectedSubgroupAssignmentsHandler := auth.AuthMiddleware(
		tokenService,
		subgroupAssignmentsHandler,
	)

	mux.Handle(
		"POST /groups/{group_id}/subgroups/{subgroup_id}/faculty",
		protectedSubgroupAssignmentsHandler,
	)

	mux.Handle(
		"GET /groups/{group_id}/subgroups/{subgroup_id}/faculty",
		protectedSubgroupAssignmentsHandler,
	)

	mux.Handle(
		"DELETE /groups/{group_id}/subgroups/{subgroup_id}/faculty/{faculty_id}",
		protectedSubgroupAssignmentsHandler,
	)

	// Health routes
	mux.HandleFunc(
		"GET /health/live",
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(
				w,
				http.StatusOK,
				map[string]string{
					"status": "alive",
				},
			)
		},
	)

	mux.HandleFunc(
		"GET /health/ready",
		func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(
				r.Context(),
				2*time.Second,
			)
			defer cancel()

			if err := database.Ping(ctx); err != nil {
				writeJSON(
					w,
					http.StatusServiceUnavailable,
					map[string]string{
						"status": "not ready",
					},
				)
				return
			}

			writeJSON(
				w,
				http.StatusOK,
				map[string]string{
					"status": "ready",
				},
			)
		},
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info(
			"HTTP server starting",
			"addr",
			cfg.HTTPAddr,
		)

		err := server.ListenAndServe()
		if err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}

		close(serverErrors)
	}()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		return err

	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("HTTP server stopped")

	return nil
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf(
			"write JSON response: %v",
			err,
		)
	}
}
