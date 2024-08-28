// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package rename implements a command to set the name
// of a list of terminals in a tree file.
package rename

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `rename [--tree <tree-name>] [-i|--input <file>]
	[-o|--output <file>] <treefile>...`,
	Short: "rename the terminals of a tree",
	Long: `
Command rename reads one or more trees in TSV format, and use a list of names
to change the terminal names of a tree.

One or more tree files must be given as arguments.

The names of the terminals to change, as well as the new names to that
terminals can be given as a file using the flag --input, or -i, or provided in
the standard input. The terminal file must be a tab-delimited file with at
least two columns, one called "old name", and other called "new name".

The resulting tree file will be printed in the standard output. Use the flag
--output, or -o, to define an output file.

By default, all trees will be processed, use the flag --tree to define a
particular tree to be modified.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var treeName string
var input string
var output string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&treeName, "tree", "", "")
	c.Flags().StringVar(&input, "input", "", "")
	c.Flags().StringVar(&input, "i", "", "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
}

func run(c *command.Command, args []string) error {
	if len(args) == 0 {
		return c.UsageError("expecting one or more tree files")
	}

	coll := timetree.NewCollection()
	for _, a := range args {
		nc, err := readCollection(a)
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

	changes, err := readNames(c.Stdin(), input)
	if err != nil {
		return err
	}

	trees := coll.Names()
	if treeName != "" {
		trees = []string{treeName}
	}

	for _, tn := range trees {
		t := coll.Tree(tn)
		if t == nil {
			continue
		}
		for old, nw := range changes {
			id, ok := t.TaxNode(old)
			if !ok {
				continue
			}

			if err := t.SetName(id, nw); err != nil {
				return fmt.Errorf("on tree %q: %v", tn, err)
			}
		}
	}

	if err := writeTrees(c.Stdout(), coll); err != nil {
		return err
	}
	return nil
}

func readCollection(name string) (*timetree.Collection, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c, err := timetree.ReadTSV(f)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", name, err)
	}
	return c, nil
}

func readNames(r io.Reader, name string) (map[string]string, error) {
	if name != "" {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else {
		name = "stdin"
	}

	tab := csv.NewReader(r)
	tab.Comma = '\t'
	tab.Comment = '#'

	header, err := tab.Read()
	if err != nil {
		return nil, fmt.Errorf("on file %q: header: %v", name, err)
	}

	fields := make(map[string]int)
	for i, h := range header {
		h = strings.ToLower(h)
		fields[h] = i
	}

	cols := []string{"new name", "old name"}
	for _, c := range cols {
		if _, ok := fields[c]; !ok {
			return nil, fmt.Errorf("on file %q: header: expecting column %q", name, c)
		}
	}

	changes := make(map[string]string)
	for {
		row, err := tab.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		ln, _ := tab.FieldPos(0)
		if err != nil {
			return nil, fmt.Errorf("on file %q: line %d: %v", name, ln, err)
		}

		f := "old name"
		old := strings.ToLower(strings.Join(strings.Fields(row[fields[f]]), " "))
		if old == "" {
			continue
		}

		f = "new name"
		nw := strings.ToLower(strings.Join(strings.Fields(row[fields[f]]), " "))
		if nw == "" {
			continue
		}
		changes[old] = nw
	}

	return changes, nil
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
