// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package importcmd implements a command to import phylogenetic trees
// from a newick file into tsv files.
package importcmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `import [--age <value>] [-o|--output <file>]
	--name <tree-name> [<newick-file>...]`,
	Short: "import a newick tree",
	Long: `
Command import reads one or more files that contain phylogenetic trees in
Newick format (i.e. parenthetical format), and import them into an equivalent
file in TSV format.

One or more newick files can be given as arguments. If no file is given the
input will be read from the standard input.

Trees in TSV format must have names. The flag --name is required and sets the
name of the tree. If multiple trees are found, the name will be append with
sequential numbers.

By default the output will be printed in the standard output. To define an
output file use the flag --output, or -o. If the file already exists, imported
trees will be added to the file.

The output TSV file will contain the following fields:

	- tree, for the name of the tree
	- node, for the ID of the node
	- parent, for the ID of the parent node
	    (-1 is used for the root)
	- age, the age of the node (in years)
	- taxon, the taxonomic name of the node

By default, the age of the tree will be calculated using the maximum branch
length between the root and its terminals. Use the flag --age to set a
different age for the root (in million years). The given age should be greater
or equal to the maximum branch length.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var output string
var age float64
var nameFlag string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
	c.Flags().StringVar(&nameFlag, "name", "", "")
	c.Flags().Float64Var(&age, "age", 0, "")
}

func run(c *command.Command, args []string) error {
	if nameFlag == "" {
		return c.UsageError("flag --name undefined")
	}

	coll, err := newTreeCollection()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		args = append(args, "-")
	}
	for i, a := range args {
		nm := nameFlag
		if i > 0 {
			nm = fmt.Sprintf("%s.%d", nameFlag, i)
		}

		nc, err := readNewick(c.Stdin(), a, nm)
		if err != nil {
			return err
		}

		for _, tn := range nc.Names() {
			t := nc.Tree(tn)
			if err := coll.Add(t); err != nil {
				return fmt.Errorf("when adding trees from %q: %v", a, err)
			}
		}
	}

	if err := writeTrees(c.Stdout(), coll); err != nil {
		return err
	}
	return nil
}

func newTreeCollection() (*timetree.Collection, error) {
	if output == "" {
		return timetree.NewCollection(), nil
	}

	f, err := os.Open(output)
	if errors.Is(err, os.ErrNotExist) {
		return timetree.NewCollection(), nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c, err := timetree.ReadTSV(f)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", output, err)
	}
	return c, nil
}

// millionYears is used transform the age flag
// (a float in million years)
// into an integer in years.
const millionYears = 1_000_000

func readNewick(r io.Reader, treeFile, name string) (*timetree.Collection, error) {
	if treeFile != "-" {
		f, err := os.Open(treeFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else {
		treeFile = "stdin"
	}

	c, err := timetree.Newick(r, name, int64(age*millionYears))
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", treeFile, err)
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
