package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/neohetj/matrix/internal/loader"
	"github.com/neohetj/matrix/pkg/validation"
)

type repeatedFlag []string

func (f *repeatedFlag) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *repeatedFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string, stdout *os.File) error {
	var moduleRoot string
	var dslRoots repeatedFlag
	flags := flag.NewFlagSet("matrix-validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&moduleRoot, "module-root", ".", "module repository root to scan")
	flags.Var(&dslRoots, "dsl-root", "DSL root relative to module-root; repeat to include multiple roots")
	if err := flags.Parse(args); err != nil {
		return err
	}

	provider := loader.NewFileProvider(moduleRoot, 50)
	paths := validation.DiscoverLoaderPaths(provider, []string(dslRoots))
	report := validation.ValidateLoaderResources(provider, paths)

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
