// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package timetree provides a representation
// of a time calibrated phylogenetic tree.
package timetree

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/exp/slices"
)

var (
	// Tree adding errors
	ErrAddNoParent     = errors.New("parent ID not in tree")
	ErrAddRepeated     = errors.New("repeated taxon name")
	ErrAddInvalidBrLen = errors.New("invalid branch length")

	// Tree validation errors
	ErrValSingleChild = errors.New("node with a single descendant")
	ErrValUnnamedTerm = errors.New("unnamed terminal")
)

// A Tree is a time calibrated phylogenetic tree,
// a set of phylogenetic nodes
// with a single common ancestor.
type Tree struct {
	name string

	nodes map[int]*node
	taxa  map[string]*node
	root  *node
}

// New returns a new phylogenetic tree with a name
// and a root at the given age in years.
func New(name string, age int64) *Tree {
	t := &Tree{
		name:  name,
		nodes: make(map[int]*node),
		taxa:  make(map[string]*node),
	}
	root := &node{
		id:  0,
		age: age,
	}
	t.nodes[root.id] = root
	t.root = root

	return t
}

// Add adds a node as child of the indicated node ID,
// using the indicated branch length in years,
// and a taxon name for the node
// (that can be empty).
// It returns the ID of the added node
// or -1 and an error.
func (t *Tree) Add(id int, brLen int64, name string) (int, error) {
	p, ok := t.nodes[id]
	if !ok {
		return -1, fmt.Errorf("%w: %d", ErrAddNoParent, id)
	}

	name = canon(name)
	if name != "" {
		if _, dup := t.taxa[name]; dup {
			return -1, fmt.Errorf("%w: %s", ErrAddRepeated, name)
		}
	}

	age := p.age - brLen
	if age < 0 {
		return -1, fmt.Errorf("%w: branch length %d greater than parent age %d", ErrAddInvalidBrLen, brLen, p.age)
	}

	n := &node{
		id:     len(t.nodes),
		parent: p,
		age:    age,
		taxon:  name,
		brLen:  brLen,
	}
	p.children = append(p.children, n)
	t.nodes[n.id] = n
	if name != "" {
		t.taxa[name] = n
	}

	return n.id, nil
}

// Age returns the age of the indicated node.
func (t *Tree) Age(id int) int64 {
	n, ok := t.nodes[id]
	if !ok {
		return 0
	}

	return n.age
}

// Children returns an slice with the IDs
// of the children of a node.
func (t *Tree) Children(id int) []int {
	n, ok := t.nodes[id]
	if !ok {
		return nil
	}
	if n.isTerm() {
		return nil
	}

	children := make([]int, 0, len(n.children))
	for _, c := range n.children {
		children = append(children, c.id)
	}
	slices.Sort(children)
	return children
}

// Name returns the name of the tree.
func (t *Tree) Name() string {
	return t.name
}

// Nodes return an slice with IDs
// of the nodes of the tree.
func (t *Tree) Nodes() []int {
	ns := make([]int, 0, len(t.nodes))
	for _, n := range t.nodes {
		ns = append(ns, n.id)
	}
	slices.Sort(ns)
	return ns
}

// Parent returns the ID of the parent
// of the indicated node.
// It will return -1 for the root or an invalid node.
func (t *Tree) Parent(id int) int {
	n, ok := t.nodes[id]
	if !ok {
		return -1
	}

	if n.parent == nil {
		return -1
	}
	return n.parent.id
}

// Root returns the ID of the root node
// which is 0.
func (t *Tree) Root() int {
	return t.root.id
}

// Taxa returns all defined taxon names of the tree.
func (t *Tree) Taxa() []string {
	taxa := make([]string, 0, len(t.taxa))
	for _, n := range t.taxa {
		taxa = append(taxa, n.taxon)
	}
	slices.Sort(taxa)
	return taxa
}

// Taxon returns the taxon name
// of the node with the indicated ID.
func (t *Tree) Taxon(id int) string {
	n, ok := t.nodes[id]
	if !ok {
		return ""
	}

	return n.taxon
}

// Terms returns the name of all terminals of the tree.
func (t *Tree) Terms() []string {
	terms := make([]string, 0, len(t.taxa))
	for _, n := range t.taxa {
		if !n.isTerm() {
			continue
		}
		terms = append(terms, n.taxon)
	}
	slices.Sort(terms)
	return terms
}

// Validate will return an error if the tree is invalid.
// A tree is invalid if it has nodes with a single child,
// or terminal nodes are without a defined name.
func (t *Tree) Validate() error {
	for _, n := range t.nodes {
		if len(n.children) == 1 {
			return fmt.Errorf("%w: %d", ErrValSingleChild, n.id)
		}
		if n.isTerm() && n.taxon == "" {
			return fmt.Errorf("%w: %d", ErrValUnnamedTerm, n.id)
		}
	}
	return nil
}

// A Node is a node in a phylogenetic tree.
type node struct {
	id     int
	parent *node
	age    int64
	taxon  string

	brLen int64

	children []*node
}

// IsTerm returns true if the node is a terminal
// (i.e. has no children).
func (n *node) isTerm() bool {
	return len(n.children) == 0
}

// Canon returns a taxon name
// in its canonical form.
func canon(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return ""
	}
	name = strings.ToLower(name)
	r, n := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(r)) + name[n:]
}
