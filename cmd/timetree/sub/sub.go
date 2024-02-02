// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package sub implements a command to produce a sub-tree
// from a phylogenetic tree in a tsv file.
package sub

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `sub [-i|--input <file>] [-o|--output <file>]
	[--name <tree-name>] --tree <tree-name>
	<taxon-1> <taxon-2> [<taxon-n>...]`,
	Short: "retrieve a sub-tree",
	Long: `
Command sub reads a tree file in TSV format and selects the clade that contains
the most recent common ancestor of the indicated terminals.

By default, the input tree will be read from the standard input. Use the flag
--input or -i to set a particular input file name.

By default, the output will be printed in the standard output. Use the flag
--output, or -o, to define an output file. If the file already exists, the
resulting tree will be added to the file.

The flag --tree is required and defines the name of the source tree.

By default, the resulting tree will be named after the name of the node; if the
node does not have a name, it will use the name of the source tree and the node
ID in that tree. Use the flag --name to define a name for the resulting tree.

The arguments of the command are the names of at least two taxons named in the
source tree; the most recent common ancestor of the indicated names will be
used as the root node for the resulting tree.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var input string
var output string
var nameFlag string
var treeFlag string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&input, "input", "", "")
	c.Flags().StringVar(&input, "i", "", "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
	c.Flags().StringVar(&nameFlag, "name", "", "")
	c.Flags().StringVar(&treeFlag, "tree", "", "")
}

func run(c *command.Command, args []string) error {
	if treeFlag == "" {
		return c.UsageError("flag --tree must be defined")
	}
	if len(args) < 2 {
		return c.UsageError("at least two taxon names must be given")
	}

	coll, err := readCollection(c.Stdin(), input)
	if err != nil {
		return err
	}
	t := coll.Tree(treeFlag)
	if t == nil {
		return fmt.Errorf("tree %q not found", treeFlag)
	}

	mrca := t.MRCA(args...)
	if mrca < 0 {
		return fmt.Errorf("most recent common ancestor of %v not found on tree %q", args, treeFlag)
	}
	nt := t.SubTree(mrca, nameFlag)

	if err := writeTrees(c.Stdout(), nt); err != nil {
		return err
	}
	return nil
}

func readCollection(r io.Reader, name string) (*timetree.Collection, error) {
	if name != "" {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
		name = "stdin"
	}

	c, err := timetree.ReadTSV(r)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", name, err)
	}
	return c, nil
}

func writeTrees(w io.Writer, t *timetree.Tree) (err error) {
	var c *timetree.Collection
	if output != "" {
		c, err = getCollection()
		if err != nil {
			return err
		}

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
	} else {
		output = "stdout"
	}

	if c == nil {
		c = timetree.NewCollection()
	}
	if err := c.Add(t); err != nil {
		return err
	}

	if err := c.TSV(w); err != nil {
		return fmt.Errorf("while writing to %q: %v", output, err)
	}
	return nil
}

func getCollection() (*timetree.Collection, error) {
	f, err := os.Open(output)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	defer f.Close()

	c, err := timetree.ReadTSV(f)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", output, err)
	}
	return c, nil
}
