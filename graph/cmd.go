package graph

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/subcommands"

	"github.com/loov/goda/pkg"
	"github.com/loov/goda/templates"
)

type Command struct {
	printStandard bool
	format        string
}

func (*Command) Name() string     { return "graph" }
func (*Command) Synopsis() string { return "Print dependency graph." }
func (*Command) Usage() string {
	return `graph <pkg>:
	Print dependency dot graph.

  `
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")
	f.StringVar(&cmd.format, "format", "{{.ID}}", "formatting")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing package names\n")
		return subcommands.ExitUsageError
	}

	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid format string\n")
		return subcommands.ExitFailure
	}

	result, err := pkg.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	if !cmd.printStandard {
		result = pkg.Subtract(result, pkg.Std())
	}

	pkgs := result.Sorted()

	fmt.Fprintf(os.Stdout, "digraph G {\n")
	fmt.Fprintf(os.Stdout, "    node [shape=rectangle];")
	fmt.Fprintf(os.Stdout, "    rankdir=LR;")
	defer fmt.Fprintf(os.Stdout, "}\n")

	for _, p := range pkgs {
		var s strings.Builder
		err := t.Execute(&s, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}

		fmt.Fprintf(os.Stdout, "    %v [label=%q];\n", escapeID(p.ID), s.String())
	}

	for _, src := range pkgs {
		for _, dst := range src.Imports {
			if _, ok := result[dst.ID]; ok {
				fmt.Fprintf(os.Stdout, "    %v -> %v;\n", escapeID(src.ID), escapeID(dst.ID))
			}
		}
	}

	return subcommands.ExitSuccess
}

func escapeID(s string) string {
	var d []byte
	for _, r := range s {
		if 'a' <= r && r <= 'z' {
			d = append(d, byte(r))
		} else if 'A' <= r && r <= 'Z' {
			d = append(d, byte(r))
		} else if '0' <= r && r <= '9' {
			d = append(d, byte(r))
		} else {
			d = append(d, '_')
		}
	}
	return string(d)
}