package main

import (
	"context"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/okrammeitei/chatgo/internal/config"
)

func main() {
	configPath := os.Getenv("CHATGO_CONFIG")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	migrationsPath := os.Getenv("CHATGO_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://migrations"
	}

	// Build postgres DSN for golang-migrate (uses postgres:// scheme)
	pgDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	m, err := migrate.New(migrationsPath, pgDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	ctx := context.Background()
	_ = ctx // migrate library doesn't accept ctx yet

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			fmt.Fprintf(os.Stderr, "migrate down: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migrations rolled back")

	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			fmt.Fprintf(os.Stderr, "version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)

	case "force":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: migrate force <version>")
			os.Exit(1)
		}
		var ver int
		if _, err := fmt.Sscan(os.Args[2], &ver); err != nil {
			fmt.Fprintf(os.Stderr, "invalid version: %v\n", err)
			os.Exit(1)
		}
		if err := m.Force(ver); err != nil {
			fmt.Fprintf(os.Stderr, "force: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("forced to version %d\n", ver)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s (up|down|version|force)\n", cmd)
		os.Exit(1)
	}
}
