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
	ErrAddNoSister     = errors.New("sister ID not in tree")
	ErrAddRootSister   = errors.New("sister ID is the root node")

	// Tree validation errors
	ErrValSingleChild = errors.New("node with a single descendant")
	ErrValUnnamedTerm = errors.New("unnamed terminal")

	// Age assignments
	ErrInvalidRootAge = errors.New("invalid root age")
	ErrOlderAge       = errors.New("age to old for node")
	ErrYoungerAge     = errors.New("age to young for node")
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

// AddSister adds a node as a sister group
// of the indicated node ID,
// using the indicated age of the added node
// and the branch length of the new branch,
// both in years,
// and the name of the added node
// (that can be empty).
// It returns the ID of the added node
// or -1 and an error.
func (t *Tree) AddSister(id int, age, brLen int64, name string) (int, error) {
	sister, ok := t.nodes[id]
	if !ok {
		return -1, fmt.Errorf("%w: ID %d", ErrAddNoSister, id)
	}
	if t.root == sister {
		return -1, fmt.Errorf("%w: ID %d", ErrAddRootSister, id)
	}
	name = canon(name)
	if name != "" {
		if _, dup := t.taxa[name]; dup {
			return -1, fmt.Errorf("%w: %s", ErrAddRepeated, name)
		}
	}

	pAge := age + brLen
	if pAge < sister.age {
		return -1, fmt.Errorf("%w: sister age %d, want %d", ErrYoungerAge, pAge, sister.age)
	}

	// Add new parent
	pp := sister.parent
	if pp.age <= pAge {
		return -1, fmt.Errorf("%w: parent age %d, want %d", ErrOlderAge, pAge, pp.age)
	}
	p := &node{
		id:     len(t.nodes),
		parent: pp,
		age:    pAge,
		brLen:  pp.age - age,
	}
	t.nodes[p.id] = p
	// replace old sister with the new parent
	for i, d := range pp.children {
		if d == sister {
			pp.children[i] = p
			break
		}
	}
	// add the sister as the first children of the new parent
	p.children = append(p.children, sister)
	sister.parent = p

	// now add the taxon
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

// Format sort the nodes of a tree,
// changing node IDs if necessary.
func (t *Tree) Format() {
	t.root.sortAllChildren()
	ns := make([]*node, 0, len(t.nodes))
	ns = t.preOrder(ns, t.root)

	nodes := make(map[int]*node, len(ns))
	for i, n := range ns {
		n.id = i
		nodes[i] = n
	}
	t.nodes = nodes
}

// IsRoot returns true if the indicated node
// is the root of the tree.
func (t *Tree) IsRoot(id int) bool {
	n, ok := t.nodes[id]
	if !ok {
		return false
	}
	return n.parent == nil
}

// IsTerm returns true if the indicated node
// is a terminal of the tree.
func (t *Tree) IsTerm(id int) bool {
	n, ok := t.nodes[id]
	if !ok {
		return false
	}
	return n.isTerm()
}

// MRCA returns the most recent common ancestor
// of two or more terminals.
func (t *Tree) MRCA(names ...string) int {
	if len(names) == 0 {
		return -1
	}

	n, ok := t.taxa[names[0]]
	if !ok {
		return -1
	}
	if len(names) == 1 {
		return n.id
	}
	var mrca []int
	for n != nil {
		mrca = append([]int{n.id}, mrca...)
		n = n.parent
	}

	for _, nm := range names[1:] {
		n, ok := t.taxa[nm]
		if !ok {
			return -1
		}
		var m []int
		for n != nil {
			m = append([]int{n.id}, m...)
			n = n.parent
		}

		for i, v := range mrca {
			if i >= len(m) {
				mrca = mrca[:i]
				break
			}
			if v != m[i] {
				mrca = mrca[:i]
				break
			}
		}
	}

	return mrca[len(mrca)-1]
}

// Move sets the age of the root node (in years),
// and updates all node ages keeping the branch lengths.
// The age of the root must be at least equal to the distance
// to the most recent terminal.
func (t *Tree) Move(age int64) error {
	if max := t.root.maxLen(); age < max {
		return fmt.Errorf("%w: age %d is smaller than %d", ErrInvalidRootAge, age, max)
	}

	t.root.age = age
	t.root.propagateAge()
	return nil
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

// Set sets the age of a node
// (in years).
func (t *Tree) Set(id int, age int64) error {
	n, ok := t.nodes[id]
	if !ok {
		return nil
	}

	if p := n.parent; p != nil && p.age < age {
		return ErrOlderAge
	}

	var max int64
	for _, c := range n.children {
		if c.age > max {
			max = c.age
		}
	}
	if max > age {
		return ErrYoungerAge
	}

	n.age = age
	return nil
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

// TaxNode returns the ID of a node
// with a given taxon name.
// It returns false if the taxon does not exists.
func (t *Tree) TaxNode(name string) (int, bool) {
	name = canon(name)
	if name == "" {
		return -1, false
	}

	n, ok := t.taxa[name]
	if !ok {
		return -1, false
	}
	return n.id, true
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

func (t *Tree) preOrder(ns []*node, n *node) []*node {
	ns = append(ns, n)
	for _, c := range n.children {
		ns = t.preOrder(ns, c)
	}
	return ns
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

// FirstTerm return the first terminal
// by alphabetical order
// found in a node.
func (n *node) firstTerm() string {
	if n.isTerm() {
		return n.taxon
	}

	term := n.children[0].firstTerm()
	for _, c := range n.children[1:] {
		t := c.firstTerm()
		if t < term {
			term = t
		}
	}
	return term
}

// IsTerm returns true if the node is a terminal
// (i.e. has no children).
func (n *node) isTerm() bool {
	return len(n.children) == 0
}

// MaxLen returns the maximum length of the sub-tree
// that descends from the given node
// (including their ancestral branch).
func (n *node) maxLen() int64 {
	var cLen int64
	for _, c := range n.children {
		l := c.maxLen()
		if l > cLen {
			cLen = l
		}
	}
	return cLen + n.brLen
}

// PropagateAge updates the age of the descendant nodes.
func (n *node) propagateAge() {
	if n.parent != nil {
		n.age = n.parent.age - n.brLen
	}
	for _, c := range n.children {
		c.propagateAge()
	}
}

// Size return the number of terminals on a node.
func (n *node) size() int {
	if n.isTerm() {
		return 1
	}
	sz := 0
	for _, c := range n.children {
		sz += c.size()
	}
	return sz
}

// SortAllChildren sorts recursively
// the list of children
// of a node.
func (n *node) sortAllChildren() {
	for _, c := range n.children {
		c.sortAllChildren()
	}
	slices.SortFunc(n.children, func(a, b *node) bool {
		szA := a.size()
		szB := b.size()
		if szA != szB {
			return szA < szB
		}

		if a.age != b.age {
			// larger ages are earlier ages
			return a.age > b.age
		}

		// search for terminals in alphabetical order
		return a.firstTerm() < b.firstTerm()
	})
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
