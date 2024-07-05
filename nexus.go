// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// Nexus reads one or more tree
// from a nexus file.
// Age set the age of the root node
// (in years)
// if age is 0,
// the age of the root node will be inferred
// from the largest branch length
// between any terminal and the root.
// Branch lengths will be interpreted as million years.
func Nexus(r io.Reader, age int64) (*Collection, error) {
	nxf := bufio.NewReader(r)
	token := &strings.Builder{}

	// header
	if _, err := readToken(nxf, token); err != nil {
		return nil, fmt.Errorf("expecting '#nexus' header: %v", err)
	}
	if t := strings.ToLower(token.String()); t != "#nexus" {
		return nil, fmt.Errorf("got %q, expecting '#nexus' header", t)
	}

	// ignore all blocks except tree block
	for {
		if _, err := readToken(nxf, token); err != nil {
			return nil, fmt.Errorf("expecting 'begin' token: %v", err)
		}
		if t := strings.ToLower(token.String()); t != "begin" {
			return nil, fmt.Errorf("got %q, expecting 'begin' block", t)
		}

		if _, err := readToken(nxf, token); err != nil {
			return nil, fmt.Errorf("expecting block name: %v", err)
		}
		block := strings.ToLower(token.String())
		if block == "trees" {
			break
		}

		if err := skipBlock(nxf, token); err != nil {
			return nil, fmt.Errorf("incomplete block %q: %v", block, err)
		}
	}

	c := NewCollection()
	var labels map[string]string
	for {
		if _, err := readToken(nxf, token); err != nil {
			return nil, fmt.Errorf("incomplete block 'trees': %v", err)
		}
		t := strings.ToLower(token.String())
		if t == "end" || t == "endblock" {
			break
		}
		if t == "translate" {
			var err error
			labels, err = readTranslate(nxf, token)
			if err != nil {
				return nil, fmt.Errorf("invalid tree block: %v", err)
			}
			continue
		}
		if t == "tree" {
			tr, err := readTreeNewick(nxf, token, age)
			if err != nil {
				return nil, fmt.Errorf("incomplete block 'trees': %v", err)
			}
			translateTree(tr, labels)
			if err := c.Add(tr); err != nil {
				return nil, fmt.Errorf("when adding tree %q: %v", tr.Name(), err)
			}
			continue
		}

		if err := skipDefinition(nxf, token); err != nil {
			return nil, fmt.Errorf("incomplete block 'characters', token %q: %v", t, err)
		}
	}

	if len(c.Names()) == 0 {
		return nil, fmt.Errorf("file without trees")
	}

	return c, nil
}

func translateTree(t *Tree, labels map[string]string) {
	if len(labels) == 0 {
		return
	}
	ids := t.Terms()
	for _, id := range ids {
		tax, ok := labels[id]
		if !ok {
			continue
		}
		nID, ok := t.TaxNode(id)
		if !ok {
			continue
		}
		t.SetName(nID, tax)
	}
}

func readTreeNewick(r *bufio.Reader, token *strings.Builder, age int64) (*Tree, error) {
	// read tree name
	if _, err := readToken(r, token); err != nil {
		return nil, fmt.Errorf("while reading tree name: %v", err)
	}
	name := strings.ToLower(token.String())
	if err := skipSpaces(r); err != nil {
		return nil, fmt.Errorf("expecting newick tree: %v", err)
	}

	t, err := newick(r, name, age)
	if err != nil {
		return nil, err
	}

	delim, err := readToken(r, token)
	if err != nil {
		return nil, fmt.Errorf("while reading tree %q: %v", name, err)
	}
	if delim != ';' {
		return nil, fmt.Errorf("while reading tree %q: unexpected delimiter %q", name, string(delim))
	}

	return t, nil
}

func readTranslate(r *bufio.Reader, token *strings.Builder) (map[string]string, error) {
	labels := make(map[string]string)
	for i := 0; ; i++ {
		if _, err := readToken(r, token); err != nil {
			return nil, fmt.Errorf("while reading tree translate labels: %v, last label read: %d", err, i)
		}

		label := token.String()
		id, err := strconv.Atoi(label)
		if err != nil {
			return nil, fmt.Errorf("while reading tree translate labels: taxon %d [%q]: %v", i+1, token.String(), err)
		}
		if id != i+1 {
			return nil, fmt.Errorf("while reading tree translate labels: taxon %d [%q]: expecting %d", i+1, token.String(), i+1)
		}

		// read taxon name
		delim, err := readToken(r, token)
		if err != nil {
			return nil, fmt.Errorf("while reading tree translate labels: taxon %d [%q]: %v", i+1, token.String(), err)
		}

		taxName := strings.ReplaceAll(token.String(), "_", " ")
		taxName = canon(taxName)

		labels[label] = taxName
		if delim == ';' {
			break
		}
	}
	return labels, nil
}

func skipBlock(r *bufio.Reader, token *strings.Builder) error {
	for {
		_, err := readToken(r, token)
		t := strings.ToLower(token.String())
		if t == "end" || t == "endblock" {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func skipDefinition(r *bufio.Reader, token *strings.Builder) error {
	for {
		delim, err := readToken(r, token)
		if delim == ';' {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func readToken(r *bufio.Reader, token *strings.Builder) (delim rune, err error) {
	token.Reset()

	if err := skipSpaces(r); err != nil {
		return 0, err
	}

	r1, _, err := r.ReadRune()
	if err != nil {
		return 0, err
	}
	if r1 == '\'' || r1 == '"' {
		// quoted block
		stop := r1
		for {
			r1, _, err := r.ReadRune()
			if err != nil {
				return 0, err
			}
			if r1 == stop {
				nx, _, err := r.ReadRune()
				if err != nil {
					return 0, err
				}
				if nx != stop {
					r.UnreadRune()
					delim = ' '
					break
				}
				if stop == '\'' {
					continue
				}
			}
			token.WriteRune(r1)
		}
	} else {
		r.UnreadRune()
		for {
			r1, _, err := r.ReadRune()
			if err != nil {
				return 0, err
			}
			if unicode.IsSpace(r1) {
				delim = ' '
				break
			}
			if r1 == ';' || r1 == ',' || r1 == '/' || r1 == '=' {
				delim = r1
				break
			}
			token.WriteRune(r1)
		}
	}

	if unicode.IsSpace(delim) {
		if err := skipSpaces(r); err != nil {
			return 0, err
		}
		r1, _, err := r.ReadRune()
		if err != nil {
			return 0, err
		}
		if r1 == ';' || r1 == ',' || r1 == '/' || r1 == '=' {
			delim = r1
		} else {
			r.UnreadRune()
		}
	}
	return delim, nil
}

func skipSpaces(r *bufio.Reader) error {
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return err
		}

		// a comment
		if r1 == '[' {
			if err := skipComment(r); err != nil {
				return err
			}
			continue
		}

		if !unicode.IsSpace(r1) {
			r.UnreadRune()
			return nil
		}
	}
}

func skipComment(r *bufio.Reader) error {
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			return err
		}

		// a comment
		if r1 == ']' {
			return nil
		}
	}
}
