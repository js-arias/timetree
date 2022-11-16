// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package list implements a command to print
// a list of trees in a tree file.
package list

import (
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: "list [<tree-file>...]",
	Short: "print a list of trees from a file",
	Long: `
Command list reads a tree file in TSV format and print the list of the tree
names in that file.

One or more tree files in TSV format can be given as arguments. If no file is
given, the trees will be read from the standard input.
	`,
	Run: run,
}

func run(c *command.Command, args []string) error {
	coll := timetree.NewCollection()

	if len(args) == 0 {
		args = append(args, "-")
	}
	for _, a := range args {
		nc, err := readCollection(c.Stdin(), a)
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

	ls := coll.Names()
	for _, tn := range ls {
		fmt.Fprintf(c.Stdout(), "%s\n", tn)
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
