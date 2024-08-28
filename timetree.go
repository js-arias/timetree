// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package timetree provides a representation
// of a time calibrated phylogenetic tree.
package timetree

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
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

// Delete removes a node
// and all of its descendants
// from a tree.
func (t *Tree) Delete(id int) error {
	n, ok := t.nodes[id]
	if !ok {
		return nil
	}

	p := n.parent

	// polytomous node
	if len(p.children) > 2 {
		// remove the node
		for i, c := range p.children {
			if c == n {
				p.children[i] = nil
				p.children = append(p.children[:i], p.children[i+1:]...)
				break
			}
		}
		n.parent = nil
		n.del(t)
		return nil
	}

	// dichotomous node
	anc := p.parent

	// the node is the most basal group.
	if anc == nil {
		for i, c := range p.children {
			if c != n {
				t.root = c
				c.parent = nil
				p.children[i] = nil
				break
			}
		}
		p.del(t)
		return nil
	}

	// remove parent node
	for i, c := range anc.children {
		if c != p {
			continue
		}
		for j, s := range p.children {
			if s == n {
				continue
			}
			anc.children[i] = s
			p.children[j] = nil
			s.parent = anc
		}
	}

	p.parent = nil
	p.del(t)
	return nil
}

// Depth returns the number of nodes between the indicated node
// and the root of the tree.
func (t *Tree) Depth(id int) int {
	n, ok := t.nodes[id]
	if !ok {
		return -1
	}

	var d int
	for n != t.root {
		d++
		n = n.parent
	}
	return d
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

// Len returns the total length
// (in years)
// of a tree.
func (t *Tree) Len() int64 {
	return t.root.totalLen()
}

// LenToRoot returns the length
// (in years)
// from a node to the root of the tree.
func (t *Tree) LenToRoot(id int) int64 {
	n, ok := t.nodes[id]
	if !ok {
		return 0
	}

	return t.root.age - n.age
}

// MRCA returns the most recent common ancestor
// of two or more terminals.
func (t *Tree) MRCA(names ...string) int {
	if len(names) == 0 {
		return -1
	}

	n, ok := t.taxa[canon(names[0])]
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
		nm = canon(nm)
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

// NumInternal returns the number of internal nodes
// (i.e., nodes with descendants).
func (t *Tree) NumInternal() int {
	num := 0
	for _, n := range t.nodes {
		if len(n.children) == 0 {
			continue
		}
		num++
	}
	return num
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

// SetName sets the name of a node,
// removing any previous name of the node.
// If the node is not a terminal,
// the new name must be non-empty.
func (t *Tree) SetName(id int, name string) error {
	n, ok := t.nodes[id]
	if !ok {
		return nil
	}

	name = canon(name)
	if name == "" {
		if n.isTerm() {
			return ErrValUnnamedTerm
		}
		if n.taxon == "" {
			return nil
		}
		delete(t.taxa, n.taxon)
		return nil
	}

	if _, dup := t.taxa[name]; dup {
		return fmt.Errorf("%w: %s", ErrAddRepeated, name)
	}

	if n.taxon != "" {
		delete(t.taxa, n.taxon)
	}
	n.taxon = name
	t.taxa[name] = n
	return nil
}

// SubTree creates a new tree from a given node
// using the indicated name.
// If no name is given,
// it will use the node name,
// or a node identifier.
func (t *Tree) SubTree(id int, name string) *Tree {
	n, ok := t.nodes[id]
	if !ok {
		return nil
	}

	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = n.taxon
	}
	if name == "" {
		name = fmt.Sprintf("%s:node-%d", t.name, id)
	}
	name = strings.ToLower(name)

	sub := &Tree{
		name:  name,
		nodes: make(map[int]*node),
		taxa:  make(map[string]*node),
	}
	root := sub.copySource(nil, n)
	sub.root = root

	sub.Format()

	return sub
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

// CopyNode copies a node
// and all of its descendants.
func (t *Tree) copySource(p *node, src *node) *node {
	n := &node{
		id:     len(t.nodes),
		parent: p,
		age:    src.age,
		taxon:  src.taxon,
	}
	t.nodes[n.id] = n
	for _, c := range src.children {
		d := t.copySource(n, c)
		n.children = append(n.children, d)
	}
	if p != nil {
		n.brLen = p.age - n.age
	}
	if n.taxon != "" {
		t.taxa[n.taxon] = n
	}

	return n
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

// Delete a node and all of its descendants.
func (n *node) del(t *Tree) {
	delete(t.nodes, n.id)
	if n.taxon != "" {
		delete(t.taxa, n.taxon)
	}

	for _, c := range n.children {
		if c == nil {
			continue
		}
		c.del(t)
	}
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
	slices.SortFunc(n.children, func(a, b *node) int {
		szA := a.size()
		szB := b.size()
		if szA != szB {
			if szA < szB {
				return -1
			}
			return 1
		}

		if a.age != b.age {
			// larger ages are earlier ages
			if a.age > b.age {
				return -1
			}
			return 1
		}

		// search for terminals in alphabetical order
		if a.firstTerm() < b.firstTerm() {
			return -1
		}
		return 1
	})
}

// TotalLen returns the length of all the branches descendant
// from a node.
func (n *node) totalLen() int64 {
	var l int64
	for _, c := range n.children {
		l += c.totalLen()
	}

	if n.parent != nil {
		l += n.parent.age - n.age
	}
	return l
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
