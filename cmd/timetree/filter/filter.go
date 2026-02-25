// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package filter implements a command to remove taxons
// from a tree.
package filter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `filter [--keep]
	[-i|--input <file>] [-o|--output <file>]	
	[--tree <tree>]
	<treefile>...`,
	Short: "filter terminals of a tree",
	Long: `
Command filter reads one or more trees in TSV format, and remove the terminals
from a list of names.

One or more tree files must be given as arguments.

The name of the terminals to remove can be given either from an input file
defined with the --input, or -i, flag, or provided in the standard input. Each
line will be interpreted as terminal, and lines starting with sharp character
('#') and empty lines will be ignored.

By default, the read names will be removed from the tree. If the flag --keep
is defined, then, only the given names will be retained.

By default, the read names will be removed in all trees. Use the flag --tree
to define a single tree from the source tree collection.

The resulting tree file will be printed in the standard output. Use the flag
--output, or -o, to define an output file.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var keep bool
var input string
var output string
var treeName string

func setFlags(c *command.Command) {
	c.Flags().BoolVar(&keep, "keep", false, "")
	c.Flags().StringVar(&input, "input", "", "")
	c.Flags().StringVar(&input, "i", "", "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
	c.Flags().StringVar(&treeName, "tree", "", "")
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

	taxList, err := readTerminals(c.Stdin())
	if err != nil {
		return err
	}
	if len(taxList) == 0 {
		return nil
	}

	trees := coll.Names()
	if treeName != "" {
		trees = []string{treeName}
	}

	filterFunc := removeTips
	if keep {
		filterFunc = keepTips
	}

	for _, tn := range trees {
		t := coll.Tree(tn)
		if t == nil {
			continue
		}
		if err := filterFunc(t, taxList); err != nil {
			return err
		}
		t.Format()
	}
	if err := writeTrees(c.Stdout(), coll); err != nil {
		return err
	}
	return nil

}

func removeTips(t *timetree.Tree, taxList []string) error {
	for _, tax := range taxList {
		id, ok := t.TaxNode(tax)
		if !ok {
			continue
		}
		if err := t.Delete(id); err != nil {
			return fmt.Errorf("tree %q: taxon %q: %v", t.Name(), tax, err)
		}
	}
	return nil
}

func keepTips(t *timetree.Tree, taxList []string) error {
	taxNames := make(map[string]bool)
	for _, tax := range taxList {
		id, ok := t.TaxNode(tax)
		if !ok {
			continue
		}
		taxNames[t.Taxon(id)] = true
	}
	if len(taxNames) == 0 {
		return nil
	}

	for _, tax := range t.Terms() {
		if taxNames[tax] {
			continue
		}
		id, ok := t.TaxNode(tax)
		if !ok {
			continue
		}
		if err := t.Delete(id); err != nil {
			return fmt.Errorf("tree %q: taxon %q: %v", t.Name(), tax, err)
		}
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

func readTerminals(r io.Reader) ([]string, error) {
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else {
		input = "stdin"
	}

	br := bufio.NewReader(r)
	var tax []string
	for i := 1; ; i++ {
		tx, err := br.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: line %d: %v", input, i, err)
		}
		tx = strings.Join(strings.Fields(tx), " ")
		if tx == "" {
			continue
		}
		if tx[0] == '#' {
			continue
		}
		tax = append(tax, tx)
	}
	return tax, nil
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
