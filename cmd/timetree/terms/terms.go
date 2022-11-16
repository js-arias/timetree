// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package terms implements a command to print
// the list of terminals in a tree file.
package terms

import (
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
	"golang.org/x/exp/slices"
)

var Command = &command.Command{
	Usage: "terms [--tree <tree-name>] [<tree-file>...]",
	Short: "print a list of tree terminals from a file",
	Long: `
Command terms reads a tree file in TSV format and print the list of the
terminals of each tree in the file.

One or more tree files in TSV format can be given as arguments. If no file is
given, the trees will be read from the standard input.

By default all terminals will be printed. If the flag --tree is set, only the
terminals of the indicated tree will be printed.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var treeName string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&treeName, "tree", "", "")
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

	ls := makeList(coll)
	for _, term := range ls {
		fmt.Fprintf(c.Stdout(), "%s\n", term)
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

func makeList(c *timetree.Collection) []string {
	if treeName != "" {
		t := c.Tree(treeName)
		if t == nil {
			return nil
		}
		return t.Terms()
	}

	terms := make(map[string]bool)
	for _, tn := range c.Names() {
		t := c.Tree(tn)
		for _, tax := range t.Terms() {
			terms[tax] = true
		}
	}

	termList := make([]string, 0, len(terms))
	for tax := range terms {
		termList = append(termList, tax)
	}
	slices.Sort(termList)

	return termList
}
