// Command dbctl verifies the feature-free Atlas migration release set.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/MichaelSeveen/atlas/internal/platform/migration"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "database control verification failed")
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 || arguments[0] != "verify" {
		return errors.New("usage: dbctl verify [--migration-dir path]")
	}
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	directory := flags.String("migration-dir", "db/migrations", "released migration directory")
	if err := flags.Parse(arguments[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected database control argument")
	}
	migrations, err := migration.Load(*directory)
	if err != nil {
		return err
	}
	latest := migrations[len(migrations)-1]
	fmt.Printf("migration_count=%d\n", len(migrations))
	fmt.Printf("current_version=%d\n", latest.Version)
	fmt.Printf("current_checksum=%s\n", latest.Checksum)
	fmt.Println("migration_manifest=PASS")
	return nil
}
