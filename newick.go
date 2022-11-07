// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

var (
	// Newick errors
	ErrNotNewick  = fmt.Errorf("not a newick tree file")
	ErrUnexpBrLen = fmt.Errorf("unexpected branch length")
)

// Newick reads one or more trees in newick (parenthetical) format.
// Age set the age of the root node
// (in years),
// if age is 0,
// the age of the root node will be inferred
// from the largest branch length
// between any terminal and the root.
// Branch lengths will be interpreted as million years.
// Name sets the name of the first tree,
// any other tree name will be
// in the form <name>.<number>
// starting from 1.
func Newick(r io.Reader, name string, age int64) (*Collection, error) {
	name = strings.ToLower(strings.Join(strings.Fields(name), " "))
	if name == "" {
		return nil, ErrTreeNoName
	}
	c := NewCollection()

	bw := bufio.NewReader(r)

	for i := 0; ; i++ {
		nm := name
		if i > 0 {
			nm = fmt.Sprintf("%s.%d", name, i)
		}
		t, err := newick(bw, nm, age)
		if err != nil {
			return nil, err
		}
		if t == nil {
			if i > 0 {
				break
			}
			return nil, ErrNotNewick
		}
		if err := c.Add(t); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func newick(r *bufio.Reader, name string, age int64) (*Tree, error) {
	// search for the first parenthesis of the tree.
	for {
		r1, _, err := r.ReadRune()
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if r1 == '(' {
			break
		}
	}

	t := &Tree{
		name:  name,
		nodes: make(map[int]*node),
		taxa:  make(map[string]*node),
	}

	last := ""
	root, err := t.readNewick(r, nil, &last)
	if err != nil {
		return nil, err
	}
	t.root = root
	max := t.root.maxLen()
	if age == 0 {
		age = max
	}
	if max > age {
		return nil, fmt.Errorf("%w: age should be greater than %d years", ErrInvalidRootAge, max)
	}
	t.root.age = age
	t.root.propagateAge()

	t.sortNodes()

	return t, nil
}

// MillionYears is used to transform newick branch lengths
// (a float in million years)
// to an integer in years.
const millionYears = 1_000_000

func (t *Tree) readNewick(r *bufio.Reader, parent *node, last *string) (*node, error) {
	n := &node{
		id:     len(t.nodes),
		parent: parent,
	}
	t.nodes[n.id] = n

	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return nil, fmt.Errorf("%v: last read terminal: %s", err, *last)
		}
		if r1 == ':' {
			return nil, fmt.Errorf("%w: last read terminal: %s", ErrUnexpBrLen, *last)
		}
		if unicode.IsSpace(r1) || r1 == ',' {
			continue
		}
		if r1 == '(' {
			// an internal node
			child, err := t.readNewick(r, n, last)
			if err != nil {
				return nil, err
			}
			n.children = append(n.children, child)
			continue
		}
		if r1 == ')' {
			break
		}
		if r1 == ';' {
			r.UnreadRune()
			break
		}

		// a terminal
		r.UnreadRune()
		term, bl, err := readTerm(r)
		if err != nil {
			if term != "" {
				*last = term
			}
			return nil, fmt.Errorf("%w: last read terminal: %s", err, *last)
		}
		if _, dup := t.taxa[term]; dup {
			return nil, fmt.Errorf("%w: %s", ErrAddRepeated, term)
		}
		child := &node{
			id:     len(t.nodes),
			parent: n,
			taxon:  term,
			brLen:  int64(bl * millionYears),
		}
		t.nodes[child.id] = child
		n.children = append(n.children, child)
		t.taxa[term] = child
		*last = term
	}

	if len(n.children) < 2 {
		return nil, fmt.Errorf("%w: last read terminal: %s", ErrValSingleChild, *last)
	}

	bl, err := readBrLen(r)
	if err != nil {
		return nil, fmt.Errorf("%w: last read terminal: %s", err, *last)
	}
	n.brLen = int64(bl * millionYears)

	return n, nil
}

// ReadBlock reads a string
// inside a quoted block.
func readBlock(r *bufio.Reader, delim rune) (string, error) {
	var b strings.Builder
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return "", err
		}
		if r1 == delim {
			break
		}
		if r1 == '(' || r1 == ')' || r1 == ':' || r1 == ',' {
			continue
		}
		b.WriteRune(r1)
	}
	return b.String(), nil
}

// ReadBrLen reads the length of the branch
// connecting the node with its ancestor.
func readBrLen(r *bufio.Reader) (float64, error) {
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return 0, err
		}

		if r1 == ':' {
			break
		}
		if r1 == ',' || unicode.IsSpace(r1) {
			return 0, nil
		}
		if r1 == '\'' {
			if _, err := readBlock(r, '\''); err != nil {
				return 0, err
			}
			continue
		}
		if r1 == '(' || r1 == ')' || r1 == ';' {
			r.UnreadRune()
			return 0, nil
		}
	}

	var b strings.Builder
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return 0, nil
		}
		if unicode.IsSpace(r1) || r1 == ',' {
			break
		}
		if r1 == '(' || r1 == ')' {
			r.UnreadRune()
			break
		}
		b.WriteRune(r1)
	}
	s := b.String()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid value %q", ErrAddInvalidBrLen, s)
	}
	if v < 0 {
		return 0, fmt.Errorf("%w: invalid value %q", ErrAddInvalidBrLen, s)
	}

	// Set 0 length branches to be equal to a year
	if v < 1.0/millionYears {
		v = 1.0 / millionYears
	}
	return v, nil
}

// ReadName reads a terminal name.
func readName(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return "", err
		}
		if unicode.IsSpace(r1) {
			break
		}
		if r1 == '(' || r1 == ')' || r1 == ':' || r1 == ',' {
			r.UnreadRune()
			break
		}
		if r1 == '_' {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r1)
	}
	return b.String(), nil
}

// ReadTerm reads a terminal name
// and its branch length
func readTerm(r *bufio.Reader) (string, float64, error) {
	r1, _, _ := r.ReadRune()

	var name string
	var err error
	if r1 == '\'' {
		name, err = readBlock(r, '\'')
	} else {
		r.UnreadRune()
		name, err = readName(r)
	}
	if err != nil {
		return "", 0, err
	}

	name = canon(name)
	if name == "" {
		return "", 0, ErrValUnnamedTerm
	}

	bl, err := readBrLen(r)
	if err != nil {
		return name, 0, err
	}
	return name, bl, nil
}
