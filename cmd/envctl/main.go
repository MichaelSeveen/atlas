// Command envctl validates and safely prepares Atlas environment configuration.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/environment"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "atlas environment command failed")
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("environment subcommand is required")
	}
	switch arguments[0] {
	case "validate":
		return validateCommand(arguments[1:])
	case "prepare":
		return prepareCommand(arguments[1:])
	case "reset":
		return resetCommand(arguments[1:])
	case "seed-checksum":
		return seedChecksumCommand(arguments[1:])
	default:
		return errors.New("unknown environment subcommand")
	}
}

func validateCommand(arguments []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	configDirectory := flags.String("config-dir", "deploy/environments", "environment configuration directory")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("invalid validate arguments")
	}
	configs := make([]environment.Config, 0, len(environment.Names()))
	for _, name := range environment.Names() {
		config, err := environment.Load(filepath.Join(*configDirectory, string(name)+".json"), time.Now().UTC())
		if err != nil {
			return err
		}
		configs = append(configs, config)
	}
	if err := environment.ValidateSet(configs); err != nil {
		return err
	}
	fmt.Printf("environment_configurations=%d\nconfiguration_validation=PASS\n", len(configs))
	return nil
}

func prepareCommand(arguments []string) error {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	name := flags.String("environment", "", "environment name")
	configDirectory := flags.String("config-dir", "deploy/environments", "environment configuration directory")
	stateRoot := flags.String("state-root", ".tmp/environments", "environment state root")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *name == "" {
		return errors.New("invalid prepare arguments")
	}
	runtimePath, err := environment.Prepare(environment.Name(*name), *configDirectory, *stateRoot, time.Now().UTC())
	if err != nil {
		return err
	}
	fmt.Printf("environment=%s\nruntime_file=%s\nprepare=PASS\n", *name, filepath.Clean(runtimePath))
	return nil
}

func resetCommand(arguments []string) error {
	flags := flag.NewFlagSet("reset", flag.ContinueOnError)
	name := flags.String("environment", "", "environment name")
	confirmation := flags.String("confirm", "", "exact reset confirmation")
	stateRoot := flags.String("state-root", ".tmp/environments", "environment state root")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *name == "" {
		return errors.New("invalid reset arguments")
	}
	target, err := environment.Reset(environment.Name(*name), *stateRoot, *confirmation)
	if err != nil {
		return err
	}
	fmt.Printf("resolved_environment=%s\nremoved_target=%s\nreset=PASS\n", *name, filepath.Clean(target))
	return nil
}

func seedChecksumCommand(arguments []string) error {
	flags := flag.NewFlagSet("seed-checksum", flag.ContinueOnError)
	manifestPath := flags.String("manifest", "deploy/seeds/foundation.json", "seed manifest")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("invalid seed-checksum arguments")
	}
	manifest, digest, err := environment.LoadSeedManifest(*manifestPath)
	if err != nil {
		return err
	}
	fmt.Printf("seed_id=%s\nseed_checksum=%s\nseed_validation=PASS\n", manifest.SeedID, digest)
	return nil
}
