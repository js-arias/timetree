// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

var headerFields = []string{
	"tree",
	"node",
	"parent",
	"age",
	"taxon",
}

// ReadTSV reads a phylogenetic tree
// from a TSV file.
//
// The TSV must contain the following fields:
//
//	-tree, for the name of the tree
//	-node, for the ID of the node
//	-parent, for of ID of the parent node
//	    (-1 is used for the root)
//	-age, the age of the node (in years)
//	-taxon, the taxonomic name of the node
//
// Parent nodes should be defined,
// before any children node.
// Terminal nodes should have a unique taxonomic name.
//
// Here is an example file:
//
//	     # time calibrated phylogenetic tree
//	     tree	node	parent	age	taxon
//		dinosaurs	0	-1	235000000
//		dinosaurs	1	0	230000000	Eoraptor lunensis
//		dinosaurs	2	0	170000000
//		dinosaurs	3	2	145000000	Ceratosaurus nasicornis
//		dinosaurs	4	2	71000000	Carnotaurus sastrei
func ReadTSV(r io.Reader) (*Collection, error) {
	tab := csv.NewReader(r)
	tab.Comma = '\t'
	tab.Comment = '#'

	head, err := tab.Read()
	if err != nil {
		return nil, fmt.Errorf("while reading header: %v", err)
	}
	fields := make(map[string]int, len(head))
	for i, h := range head {
		h = strings.ToLower(h)
		fields[h] = i
	}
	for _, h := range headerFields {
		if _, ok := fields[h]; !ok {
			return nil, fmt.Errorf("expecting field %q", h)
		}
	}

	c := NewCollection()
	for {
		row, err := tab.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		ln, _ := tab.FieldPos(0)
		if err != nil {
			return nil, fmt.Errorf("on row %d: %v", ln, err)
		}

		f := "tree"
		name := strings.ToLower(strings.Join(strings.Fields(row[fields[f]]), " "))
		if name == "" {
			continue
		}

		t, ok := c.trees[name]
		if !ok {
			t = &Tree{
				name:  name,
				nodes: make(map[int]*node),
				taxa:  make(map[string]*node),
			}
			c.trees[name] = t
		}

		f = "node"
		id, err := strconv.Atoi(row[fields[f]])
		if err != nil {
			return nil, fmt.Errorf("on row %d: field %q: %v", ln, f, err)
		}
		if _, dup := t.nodes[id]; dup {
			return nil, fmt.Errorf("on row %d: field %q: node ID %d already used", ln, f, id)
		}

		f = "parent"
		pID, err := strconv.Atoi(row[fields[f]])
		if err != nil {
			return nil, fmt.Errorf("on row %d: field %q: %v", ln, f, err)
		}
		var p *node
		if pID >= 0 {
			var ok bool
			p, ok = t.nodes[pID]
			if !ok {
				if err != nil {
					return nil, fmt.Errorf("on row %d: field %q: %w: %d", ln, f, ErrAddNoParent, pID)
				}
			}
		} else if t.root != nil {
			return nil, fmt.Errorf("on row %d: field %q: root already defined", ln, f)
		}

		f = "age"
		age, err := strconv.ParseInt(row[fields[f]], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("on row %d: field %q: %v", ln, f, err)
		}
		if p != nil && p.age < age {
			return nil, fmt.Errorf("on row %d: field %q: age should be less than %d", ln, f, p.age)
		}

		f = "taxon"
		tax := canon(row[fields[f]])
		if tax != "" {
			if _, dup := t.taxa[tax]; dup {
				return nil, fmt.Errorf("on row %d: field %q: %w: %s", ln, f, ErrAddRepeated, tax)
			}
		}

		n := &node{
			id:     id,
			parent: p,
			age:    age,
			taxon:  tax,
		}
		t.nodes[id] = n
		if p != nil {
			p.children = append(p.children, n)
			n.brLen = p.age - n.age
		} else {
			t.root = n
		}
		if n.taxon != "" {
			t.taxa[n.taxon] = n
		}
	}

	for _, t := range c.trees {
		t.sortNodes()
		if err := t.Validate(); err != nil {
			return nil, fmt.Errorf("tree %s: %w", t.name, err)
		}
	}

	return c, nil
}

// TSV encodes a collection of phylogenetic trees
// into a TSV file.
func (c *Collection) TSV(w io.Writer) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "# time calibrated phylogenetic trees\n")
	fmt.Fprintf(bw, "# data saved on: %s\n", time.Now().Format(time.RFC3339))
	tab := csv.NewWriter(bw)
	tab.Comma = '\t'
	tab.UseCRLF = true

	if err := tab.Write(headerFields); err != nil {
		return fmt.Errorf("while writing header: %v", err)
	}

	for _, nm := range c.Names() {
		if err := c.trees[nm].tsv(tab); err != nil {
			return fmt.Errorf("while writing data: %v", err)
		}
	}

	tab.Flush()
	if err := tab.Error(); err != nil {
		return fmt.Errorf("while writing data: %v", err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("while writing data: %v", err)
	}
	return nil
}

// TSV encodes a phylogenetic tree
// into a TSV file.
func (t *Tree) tsv(w *csv.Writer) error {
	if err := t.root.tsv(w, t.name); err != nil {
		return err
	}
	return nil
}

func (n *node) tsv(w *csv.Writer, name string) error {
	p := "-1"
	if n.parent != nil {
		p = strconv.Itoa(n.parent.id)
	}
	row := []string{
		name,
		strconv.Itoa(n.id),
		p,
		strconv.FormatInt(n.age, 10),
		n.taxon,
	}
	if err := w.Write(row); err != nil {
		return err
	}

	for _, c := range n.children {
		if err := c.tsv(w, name); err != nil {
			return err
		}
	}
	return nil
}
