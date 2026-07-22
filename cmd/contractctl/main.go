package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MichaelSeveen/atlas/internal/contractcompat"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: contractctl lint <contract...> | compare --baseline <path> --candidate <path>")
	}
	switch os.Args[1] {
	case "lint":
		if len(os.Args) < 3 {
			fail("lint requires at least one contract")
		}
		for _, path := range os.Args[2:] {
			if err := contractcompat.Lint(path); err != nil {
				fail(err.Error())
			}
			fmt.Printf("contract_lint=%s:PASS\n", path)
		}
	case "compare":
		flags := flag.NewFlagSet("compare", flag.ExitOnError)
		baseline := flags.String("baseline", "", "baseline contract path")
		candidate := flags.String("candidate", "", "candidate contract path")
		if err := flags.Parse(os.Args[2:]); err != nil {
			fail(err.Error())
		}
		if *baseline == "" || *candidate == "" {
			fail("compare requires --baseline and --candidate")
		}
		if err := contractcompat.Compare(*baseline, *candidate); err != nil {
			fail(err.Error())
		}
		fmt.Println("contract_compatibility=PASS")
	default:
		fail("unknown command " + os.Args[1])
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
