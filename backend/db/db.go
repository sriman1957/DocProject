package db

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"docproject/backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps the PostgreSQL connection pool.
type Pool struct {
	*pgxpool.Pool
}

func newPoolConfig(cfg config.Config) (*pgxpool.Config, error) {
	connString := (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, fmt.Sprint(cfg.DBPort)),
		Path:   cfg.DBName,
	}).String()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 0
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	return poolConfig, nil
}

// New creates a connection pool and verifies database connectivity.
func New(cfg config.Config) (*Pool, error) {
	poolConfig, err := newPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(
		context.Background(),
		poolConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// Close releases all connections held by the pool.
func (p *Pool) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}
