// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package tax implements a command to validate the terminal names
// for a list of trees.
package tax

import (
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/gbifer/taxonomy"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `tax [--taxonomy <file>] [--set]
	[-o|--output <file>] <treefile>...`,
	Short: "validate terminal names of a tree",
	Long: `
Command tax reads one or more trees in TSV format and uses a taxonomy to
validate the names of the terminals in the tree.

One or more tree files must be given as arguments.
	
The taxonomy file can be defined either with the flag --taxonomy or provided
in the standard input. This file is a TSV file with the following columns:

	- name      the name of the taxon
	- taxonKey  a numeric identifier for the taxon (e.g., a GBIF ID)
	- rank      the taxonomic rank of the taxon. Valid ranks are: kingdom,
	            phylum, class, order, family, genus, species, and
		    unranked.
	- status    the taxonomic status of the taxon
	- parent    the ID of the parent taxon
	
To be valid, a taxon must have "accepted" status, and with a valid rank
(different from unranked).

By default, matches with synonym names will be reported to the standard error.
Use the flag --set to change the name of the terminal to the accepted name
from the taxonomy.
	
The resulting tree file will be printed on the standard output. Use the
--output, or -o flag, to define an output file.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var setFlag bool
var taxFile string
var output string

func setFlags(c *command.Command) {
	c.Flags().BoolVar(&setFlag, "set", false, "")
	c.Flags().StringVar(&taxFile, "taxonomy", "", "")
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

	tx, err := readTaxonomy(c.Stdin())
	if err != nil {
		return err
	}

	for _, tn := range coll.Names() {
		t := coll.Tree(tn)
		if err := validateTree(c.Stderr(), t, tx); err != nil {
			return err
		}
	}

	if setFlag {
		if err := writeTrees(c.Stdout(), coll); err != nil {
			return err
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

func readTaxonomy(r io.Reader) (*taxonomy.Taxonomy, error) {
	if taxFile != "" {
		f, err := os.Open(taxFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else {
		taxFile = "stdin"
	}

	tx, err := taxonomy.Read(r)
	if err != nil {
		return nil, fmt.Errorf("on file %q: %v", taxFile, err)
	}
	return tx, nil
}

func validateTree(w io.Writer, t *timetree.Tree, tx *taxonomy.Taxonomy) error {
	ls := t.Terms()

	absent := make(map[string]bool)
	ambiguous := make(map[string][]int64)
	match := make(map[int64][]string)

	for _, n := range ls {
		ids := tx.ByName(n)
		if len(ids) == 0 {
			absent[n] = true
			continue
		}
		id := tx.AcceptedAndRanked(ids[0]).ID

		if len(ids) > 1 {
			var amb []int64
			for _, v := range ids {
				x := tx.AcceptedAndRanked(v).ID
				if x != id {
					amb = append(amb, v)
				}
			}
			if len(amb) > 0 {
				amb = append([]int64{id}, amb...)
				ambiguous[n] = amb
				continue
			}
		}

		match[id] = append(match[id], n)
	}

	mult := false
	for id, m := range match {
		if len(m) == 1 {
			continue
		}
		if !mult {
			fmt.Fprintf(w, "%s: Multiple matches:\n", t.Name())
		}
		tax := tx.Taxon(id)
		fmt.Fprintf(w, "\t%s [tax:%d]:\n", tax.Name, tax.ID)

		for _, n := range m {
			id, _ := t.TaxNode(n)
			fmt.Fprintf(w, "\t\t%s [%d]\n", n, id)
		}

		mult = true
	}

	diff := false
	for id, m := range match {
		if len(m) != 1 {
			continue
		}
		term := taxonomy.Canon(m[0])
		tax := tx.Taxon(id)
		if tax.Name == term {
			continue
		}
		tID, _ := t.TaxNode(term)

		if setFlag {
			if err := t.SetName(tID, tax.Name); err != nil {
				return err
			}
			continue
		}

		if !diff {
			fmt.Fprintf(w, "%s: Match with different name:\n", t.Name())
		}
		fmt.Fprintf(w, "\tin tree %q [%d],\n\t\tin taxonomy %q\n", term, tID, tax.Name)
		diff = true
	}

	if len(ambiguous) > 0 {
		fmt.Fprintf(w, "%s: Ambiguos names:\n", t.Name())
		for n, ids := range ambiguous {
			tID, _ := t.TaxNode(n)
			fmt.Fprintf(w, "\t%s [%d]\n", n, tID)
			for _, id := range ids {
				fmt.Fprintf(w, "\t\ttax:%d\n", id)
			}
		}
	}

	if len(absent) > 0 {
		fmt.Fprintf(w, "%s: Not in taxonomy:\n", t.Name())
		for n := range absent {
			id, _ := t.TaxNode(n)
			fmt.Fprintf(w, "\t%s [%d]\n", n, id)
		}
	}

	return nil
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
