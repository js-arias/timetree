// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package add implements a command to add a new taxon
// to a tree.
package add

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `add [-o|--output <file>]
	--tree <tree> --branch <number> --sister <id>
	<taxon-name> <age> [<treefile>]`,
	Short: "add a new taxon to a tree",
	Long: `
Command add adds a new taxon, with a given age, as a sister of the indicated
node.

The first argument of the command is the name of the taxon, if the name has
multiple words, enclosed it in quotations, for example: "Homo sapiens".

The second argument is the age to be given to the taxon, in million years.

The name of a tree file can be given as a third argument. If no file is given
it will read the tree collection from the standard input.

The flag --tree is required and indicates the name of the tree to be modified.

The flag --brLen is required and indicates the branch length, in million
years, of the branch that end at the added taxon.

The flag --sister is required and is the ID of the node that will be the
sister of the added node.

The resulting tree will be printed as a tree file in the standard output. Use
the flag --output, or -o, to define an output file. As this command modifies
the tree, it is possible that node IDs will be modified in the process.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var output string
var treeName string
var sister int
var brLen float64

func setFlags(c *command.Command) {
	c.Flags().Float64Var(&brLen, "branch", 0, "")
	c.Flags().StringVar(&treeName, "tree", "", "")
	c.Flags().IntVar(&sister, "sister", -1, "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
}

const millionYears = 1_000_000

func run(c *command.Command, args []string) error {
	if len(args) < 2 {
		return c.UsageError("expecting taxon name and age")
	}

	toAdd := args[0]
	a, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("on --age flag %q: %v", args[1], err)
	}
	age := int64(a * millionYears)

	if treeName == "" {
		return c.UsageError("--tree flag must be defined")
	}
	if sister < 0 {
		return c.UsageError("--sister flag must be defined")
	}
	if brLen <= 0 {
		return c.UsageError("--branch flag must be defined")
	}

	in := "-"
	if len(args) > 2 {
		in = args[2]
	}
	tc, err := readCollection(c.Stdin(), in)
	if err != nil {
		return err
	}

	t := tc.Tree(treeName)
	if t == nil {
		return fmt.Errorf("tree %q not found", treeName)
	}

	if _, err := t.AddSister(sister, age, int64(brLen*millionYears), toAdd); err != nil {
		return err
	}
	t.Format()

	if err := writeTrees(c.Stdout(), tc); err != nil {
		return err
	}
	return nil
}

func readCollection(r io.Reader, name string) (*timetree.Collection, error) {
	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else {
		name = "stdin"
	}

	c, err := timetree.ReadTSV(r)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", name, err)
	}
	return c, nil
}

func writeTrees(w io.Writer, c *timetree.Collection) (err error) {
	outName := "stdout"
	if output != "" {
		outName = output
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer func() {
			e := f.Close()
			if e != nil && err == nil {
				err = e
			}
		}()
		w = f
	}

	if err := c.TSV(w); err != nil {
		return fmt.Errorf("while writing to %q: %v", outName, err)
	}
	return nil
}
