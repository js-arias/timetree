// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package newick implements a command to output a phylogenetic tree
// from a TSV file into an equivalent Newick file.
package newick

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `newick [--tree <tree>] [-o|--output <file>]
	[<tree-file>...]`,
	Short: "writes a tree in newick format",
	Long: `
Command newick reads a tree in TSV format and write it into a newick
(parenthetical) text format.

One or more tree files in TSV format can be given as arguments. If no file is
given, the trees will be read from the standard input.

By default, all trees will be printed in the output. If the flag --tree is
set, only the indicated tree will be exported.

By default the output will be printed in the standard output. To define an
output file use the flag --output, or -o.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var treeName string
var output string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&treeName, "tree", "", "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
}

func run(c *command.Command, args []string) (err error) {
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

	var names []string
	if treeName != "" {
		names = []string{treeName}
	} else {
		names = coll.Names()
	}

	w := c.Stdout()
	if output != "" {
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
	} else {
		output = "stdout"
	}
	bw := bufio.NewWriter(w)

	for _, tn := range names {
		t := coll.Tree(tn)
		writeNode(bw, t, t.Root())
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("while writing to %q: %v", output, err)
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

// millionYears is used to transform branch lengths
// (an integer in years)
// to a float in million years.
const millionYears = 1_000_000

func writeNode(w io.Writer, t *timetree.Tree, node int) {
	p := t.Parent(node)
	children := t.Children(node)
	if len(children) == 0 {
		brLen := float64(t.Age(p)-t.Age(node)) / millionYears
		name := strings.Join(strings.Fields(t.Taxon(node)), "_")
		fmt.Fprintf(w, "%s:%.6f", name, brLen)
		return
	}

	// an internal node
	fmt.Fprintf(w, "(")
	for i, c := range children {
		if i > 0 {
			fmt.Fprintf(w, ", ")
		}
		writeNode(w, t, c)
	}

	if p < 0 {
		// the root
		fmt.Fprintf(w, ");\n")
		return
	}
	brLen := float64(t.Age(p)-t.Age(node)) / millionYears
	fmt.Fprintf(w, "):%.6f", brLen)
}
