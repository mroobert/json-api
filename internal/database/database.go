// Package database provides support for access the database.
package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const UniqueViolation = "23505"

// Config represents configuration properties for using the database.
//
// Notes for configuring the connection pool:
//
// 1. As a rule of thumb, a MaxOpenConns value should be set explicitly. This should be
// comfortably below any hard limits on the number of connections imposed by the
// database and infrastructure, and maybe we can think keeping it fairly low
// to act as a rudimentary throttle.Ideally we should tweak this value based on the
// results of benchmarking and load-testing.
//
// 2. In general, higher MaxOpenConns and MaxIdleConns values will lead to better performance.
// But the returns are diminishing, and you should be aware that having a too-large idle connection
// pool (with connections that are not frequently re-used) can actually lead to reduced performance
// and unnecessary resource consumption.
//
// 3. To mitigate the risk from point 2 above, you should generally set a MaxConnIdleTime value
// to remove idle connections that haven’t been used for a long time.
//
// 4. It’s probably OK to leave MaxConnLifetime as unlimited, unless your database imposes a hard
// limit on connection lifetime, or you need it specifically to facilitate something like gracefully
// swapping databases.
type Config struct {
	DSN             string
	MaxOpenConns    int    // limit on the number of ‘open’ connections (in-use + idle connections)
	MaxIdleConns    int    // limit on the number of idle connections
	MaxConnIdleTime string // sets the maximum length of time that a connection can be idle for before it is marked as expired
}

// OpenConnection knows how to open a database connection based on the configuration.
func OpenConnection(cfg Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		return nil, err
	}
	pool.Config().MaxConns = int32(cfg.MaxOpenConns)
	duration, err := time.ParseDuration(cfg.MaxConnIdleTime)
	if err != nil {
		return nil, err
	}
	pool.Config().MaxConnIdleTime = duration

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
