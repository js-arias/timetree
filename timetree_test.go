// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/js-arias/timetree"
)

type treeTest struct {
	name string
	in   string
	age  int64

	nodes []node
	terms []string
	taxa  []string
}

type node struct {
	id       int
	parent   int
	age      int64
	taxon    string
	children []int
}

func getNode(t *timetree.Tree, id int) node {
	return node{
		id:       id,
		parent:   t.Parent(id),
		age:      t.Age(id),
		taxon:    t.Taxon(id),
		children: t.Children(id),
	}
}

func TestTree(t *testing.T) {
	tree := timetree.New("test", 6_300_000)
	tt := treeTest{
		name: "test",
		nodes: []node{
			{
				id:       0,
				parent:   -1,
				age:      6_300_000,
				children: []int{1, 2},
			},
		},
	}

	nodes := []node{
		{
			id:     1,
			parent: 0,
			taxon:  "Pan",
		},
		{
			id:       2,
			parent:   0,
			age:      500_000,
			taxon:    "Homo",
			children: []int{3, 4},
		},
		{
			id:     3,
			parent: 2,
			taxon:  "Homo sapiens",
		},
		{
			id:     4,
			parent: 2,
			age:    50_000,
			taxon:  "Homo neanderthalensis",
		},
	}
	// Add nodes
	for _, n := range nodes {
		pAge := tree.Age(n.parent)
		id, _ := tree.Add(n.parent, pAge-n.age, n.taxon)

		if id != n.id {
			t.Errorf("when adding nodes: got added ID %d, want %d", id, n.id)
		}
	}

	tt.nodes = append(tt.nodes, nodes...)
	tt.taxa = []string{"Homo", "Homo neanderthalensis", "Homo sapiens", "Pan"}
	tt.terms = []string{"Homo neanderthalensis", "Homo sapiens", "Pan"}
	testTree(t, tree, tt)

	// add nodes
	// while updating the age of the root as needed.
	t0 := timetree.New("from 0", 0)
	for _, n := range nodes {
		pAge := tree.Age(n.parent)
		brLen := pAge - n.age

		zAge := t0.Age(n.parent)
		zLen := zAge - n.age

		if brLen > zLen {
			age := t0.Age(t0.Root()) + brLen - zLen
			t0.Move(age)
			pAge = tree.Age(n.parent)
			brLen = pAge - n.age
		}
		t0.Add(n.parent, brLen, n.taxon)
	}
	tt.name = "from 0"
	testTree(t, t0, tt)
}

func TestTreeErrors(t *testing.T) {
	tests := map[string]struct {
		parent int
		brLen  int64
		name   string
		err    error
	}{
		"bad parent": {
			parent: 34545,
			brLen:  5_000_000,
			name:   "Rhedosaurus",
			err:    timetree.ErrAddNoParent,
		},
		"repeated taxon": {
			parent: 0,
			brLen:  10_000,
			name:   "homo",
			err:    timetree.ErrAddRepeated,
		},
		"invalid age": {
			parent: 0,
			brLen:  135_000_000,
			name:   "Pan",
			err:    timetree.ErrAddInvalidBrLen,
		},
	}
	tree := timetree.New("test", 6_300_000)
	tree.Add(0, 500_000, "Homo")

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := tree.Add(test.parent, test.brLen, test.name)
			if !errors.Is(err, test.err) {
				t.Errorf("%s: got error '%v', want '%v'", name, err, test.err)
			}
		})
	}

	// Validation errors
	err := tree.Validate()
	if !errors.Is(err, timetree.ErrValSingleChild) {
		t.Errorf("single child: got error %v, want %v", err, timetree.ErrValSingleChild)
	}

	tree.Add(0, 6_300_000, "")
	err = tree.Validate()
	if !errors.Is(err, timetree.ErrValUnnamedTerm) {
		t.Errorf("unnamed term: got error %v, want %v", err, timetree.ErrValUnnamedTerm)
	}
}

func testTree(t testing.TB, tree *timetree.Tree, test treeTest) {
	t.Helper()

	if err := tree.Validate(); err != nil {
		t.Fatalf("%s: unexpected error: %v", test.name, err)
	}

	if nm := tree.Name(); nm != test.name {
		t.Errorf("%s: tree name: got %q", test.name, nm)
	}
	if tree.Root() != 0 {
		t.Errorf("%s: tree root ID %d, want %d", test.name, tree.Root(), 0)
	}

	nodes := tree.Nodes()
	if len(nodes) != len(test.nodes) {
		t.Fatalf("%s: got %d nodes, want %d", test.name, len(nodes), len(test.nodes))
	}

	for i, id := range nodes {
		n := getNode(tree, id)
		w := test.nodes[i]
		if !reflect.DeepEqual(n, w) {
			t.Errorf("%s: node %d: got %v, want %v", test.name, id, n, w)
		}
	}

	if len(test.taxa) > 0 {
		taxa := tree.Taxa()
		if !reflect.DeepEqual(taxa, test.taxa) {
			t.Errorf("%s: got %v taxa, want %v", test.name, taxa, test.taxa)
		}
	}

	if len(test.terms) > 0 {
		terms := tree.Terms()
		if !reflect.DeepEqual(terms, test.terms) {
			t.Errorf("%s: got %v terminals, want %v", test.name, terms, test.terms)
		}
	}
}
