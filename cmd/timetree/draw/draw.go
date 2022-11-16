// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package draw implements a command to output a phylogenetic tree
// from a TSV file into an SVG file.
package draw

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `draw [--step <value>] [--tree <tree>]
	[-o|--output <out-file>] [<tree-file>...]`,
	Short: "draw a tree into an SVG file",
	Long: `
Command draw reads a tree in TSV format and draw the tree into an SVG encoded
file.

One or more tree files in TSV format can be given as arguments. If no file is
given, the trees will be read from the standard input.

By default all trees will be drawn. If the flag --tree is set, only the
indicated tree will be printed.

The output file will be the name of each tree. If the flag --output, or -o, is
defined, the indicated name will be used as the prefix for the output files.

By default, 10 pixels units will be used per million year, use the flag --step
to define a different value (it can have decimal points).
	`,
	SetFlags: setFlags,
	Run:      run,
}

var stepX float64
var treeName string
var output string

func setFlags(c *command.Command) {
	c.Flags().Float64Var(&stepX, "step", 10, "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
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

	var names []string
	if treeName != "" {
		names = []string{treeName}
	} else {
		names = coll.Names()
	}

	for _, tn := range names {
		t := coll.Tree(tn)
		if err := writeSVG(tn, copyTree(t, stepX)); err != nil {
			return err
		}
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

func writeSVG(name string, t svgTree) (err error) {
	if output != "" {
		name = fmt.Sprintf("%s-%s.svg", output, name)
	} else {
		name += ".svg"
	}

	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		e := f.Close()
		if e != nil && err == nil {
			err = e
		}
	}()

	bw := bufio.NewWriter(f)
	if err := t.draw(bw); err != nil {
		return fmt.Errorf("while writing file %q: %v", name, err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("while writing file %q: %v", name, err)
	}
	return nil
}
