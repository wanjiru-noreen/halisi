package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/wanjiru-noreen/halisi/internal/config"
	"github.com/wanjiru-noreen/halisi/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down" && args[0] != "version") {
		return errors.New("usage: go run ./cmd/migrate [up|down|version]")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	m, err := database.NewMigrator(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	switch args[0] {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "version":
		version, dirty, versionErr := m.Version()
		if errors.Is(versionErr, migrate.ErrNilVersion) {
			fmt.Println("No migrations applied")
			return nil
		}
		if versionErr != nil {
			return errors.New("migrations: cannot read schema version")
		}
		fmt.Printf("version=%d dirty=%t\n", version, dirty)
		return nil
	}
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("No migration changes needed")
		return nil
	}
	if err != nil {
		// Driver errors can include SQL data or connection credentials.
		return errors.New("migrations: operation failed; check connectivity, migration SQL and dirty schema version")
	}
	fmt.Println("Migration completed")
	return nil
}
