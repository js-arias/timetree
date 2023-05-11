// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree_test

import (
	"errors"
	"reflect"
	"strings"
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

		r := tree.IsRoot(id)
		if n.parent == -1 && !r {
			t.Errorf("%s: is root (node %d) false", test.name, id)
		}
		if n.parent >= 0 && r {
			t.Errorf("%s: is root (node %d) true", test.name, id)
		}

		it := tree.IsTerm(id)
		if it && len(n.children) > 0 {
			t.Errorf("%s: is term (node %d) true", test.name, id)
		}
		if !it && len(n.children) == 0 {
			t.Errorf("%s: is term (node %d) false", test.name, id)
		}

		if w.taxon == "" {
			continue
		}
		term, ok := tree.TaxNode(w.taxon)
		if !ok {
			t.Errorf("%s: taxon %q: not found", test.name, w.taxon)
			continue
		}
		if term != id {
			t.Errorf("%s: taxon %q: got ID %d, want %d\n", test.name, w.taxon, term, id)
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

var dinoTree = `# some dinosaurs
tree	node	parent	age	taxon
dinos	0	-1	235000000	
dinos	1	0	230000000	Eoraptor lunensis
dinos	2	0	230000000	
dinos	3	2	170000000	
dinos	4	3	145000000	Ceratosaurus nasicornis
dinos	5	3	71000000	Carnotaurus sastrei
dinos	6	2	170000000	
dinos	7	6	68000000	Tyrannosaurus rex
dinos	8	6	160000000	
dinos	9	8	150000000	Archaeopteryx lithographica
dinos	10	8	0	Passer domesticus
`

func TestMRCA(t *testing.T) {
	tests := map[string]struct {
		terms []string
		mrca  int
	}{
		"two": {
			terms: []string{"Passer domesticus", "Ceratosaurus nasicornis"},
			mrca:  2,
		},
		"three": {
			terms: []string{"Passer domesticus", "Archaeopteryx lithographica", "Ceratosaurus nasicornis"},
			mrca:  2,
		},
		"root": {
			terms: []string{"Passer domesticus", "Eoraptor lunensis", "Ceratosaurus nasicornis"},
			mrca:  0,
		},
		"not in tree": {
			terms: []string{"Passer domesticus", "Homo sapiens"},
			mrca:  -1,
		},
		"empty": {
			mrca: -1,
		},
		"single": {
			terms: []string{"Passer domesticus"},
			mrca:  10,
		},
	}
	c, err := timetree.ReadTSV(strings.NewReader(dinoTree))
	if err != nil {
		t.Fatalf("mrca: unexpected error: %v", err)
	}

	d := c.Tree("dinos")
	if d == nil {
		t.Fatalf("mrca: tree %q not found", "dinos")
	}

	for n, test := range tests {
		mrca := d.MRCA(test.terms...)
		if mrca != test.mrca {
			t.Errorf("mrca %q: got %d, want %d", n, mrca, test.mrca)
		}
	}
}

func TestSet(t *testing.T) {
	c, err := timetree.ReadTSV(strings.NewReader(dinoTree))
	if err != nil {
		t.Fatalf("mrca: unexpected error: %v", err)
	}

	d := c.Tree("dinos")
	if d == nil {
		t.Fatalf("mrca: tree %q not found", "dinos")
	}

	if err := d.Set(8, 166_873_534); err != nil {
		t.Errorf("set: unexpected error: %v", err)
	}
	if d.Age(8) != 166_873_534 {
		t.Errorf("set: got %d, want %d", d.Age(8), 166_873_534)
	}
}

func TestSetError(t *testing.T) {
	tests := map[string]struct {
		n   int
		age int64
		err error
	}{
		"older age": {
			n:   8,
			age: 171_000_000,
			err: timetree.ErrOlderAge,
		},
		"younger age": {
			n:   8,
			age: 149_999_000,
			err: timetree.ErrYoungerAge,
		},
		"younger root": {
			n:   0,
			age: 229_000_000,
			err: timetree.ErrYoungerAge,
		},
	}

	c, err := timetree.ReadTSV(strings.NewReader(dinoTree))
	if err != nil {
		t.Fatalf("mrca: unexpected error: %v", err)
	}

	d := c.Tree("dinos")
	if d == nil {
		t.Fatalf("mrca: tree %q not found", "dinos")
	}

	for name, test := range tests {
		if err := d.Set(test.n, test.age); !errors.Is(err, test.err) {
			t.Errorf("set error %q: got error %q, want %q", name, err, test.err)
		}
	}
}
