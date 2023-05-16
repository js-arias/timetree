// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package set implements a command to set node ages
// for a list of trees.
package set

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `set [--tozero]  [-i|--input <file>]
	[-o|--output <file>] <treefile>...`,
	Short: "set ages of the nodes of a tree",
	Long: `
Command set reads one or more trees in TSV format, and use a list of node ages
to set the ages of the nodes of a tree.

One or more tree files must be given as arguments.

The ages of the nodes can be defined either from an input file defined with
the --input, or -i, flag, or provided in the standard input. The ages file is
a TSV file without header, and the following columns:

	-tree  the name of the tree
	-node  the ID of the node to set
	-age   the age (in million years) of the node

The node ages must be consistent with any other age already defined on the
tree. The changes are made sequentially.

As an usual operation is to set ages of all terminals to 0 (present), the flag
--tozero is provided to automatize this action. Note that the flag will set
all terminals in the tree collection.

The resulting tree file will be printed in the standard output. Use the flag
--output, or -o, to define an output file.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var toZero bool
var input string
var output string

func setFlags(c *command.Command) {
	c.Flags().BoolVar(&toZero, "tozero", false, "")
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

	if toZero {
		termsToZero(coll)
	} else if err := readAges(c.Stdin(), coll); err != nil {
		return err
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

const millionYears = 1_000_000

func readAges(r io.Reader, c *timetree.Collection) error {
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	} else {
		input = "stdin"
	}

	tab := csv.NewReader(r)
	tab.Comma = '\t'
	tab.Comment = '#'

	fields := map[string]int{
		"tree": 0,
		"node": 1,
		"age":  2,
	}
	for {
		row, err := tab.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		ln, _ := tab.FieldPos(0)
		if err != nil {
			return fmt.Errorf("%q: on row %d: %v", input, ln, err)
		}
		if len(row) < len(fields) {
			return fmt.Errorf("%q: got %d rows, want %d", input, len(row), len(fields))
		}

		f := "tree"
		name := strings.ToLower(strings.Join(strings.Fields(row[fields[f]]), " "))
		if name == "" {
			continue
		}

		t := c.Tree(name)
		if t == nil {
			continue
		}
		f = "node"
		id, err := strconv.Atoi(row[fields[f]])
		if err != nil {
			return fmt.Errorf("%q: on row %d: field %q: %v", input, ln, f, err)
		}
		f = "age"
		ageF, err := strconv.ParseFloat(row[fields[f]], 64)
		if err != nil {
			return fmt.Errorf("%q: on row %d: field %q: %v", input, ln, f, err)
		}

		age := int64(ageF * millionYears)
		if err := t.Set(id, age); err != nil {
			return fmt.Errorf("%q: on row %d: %v", input, ln, err)
		}
	}
	return nil
}

func termsToZero(c *timetree.Collection) {
	for _, tn := range c.Names() {
		t := c.Tree(tn)
		for _, n := range t.Terms() {
			v, _ := t.TaxNode(n)
			t.Set(v, 0)
		}
	}
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
