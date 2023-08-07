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

func TestNewick(t *testing.T) {
	tests := map[string]treeTest{
		"ultrametric": {
			name: "ultrametric",
			in:   "(Gallus_gallus:324,(Macropus_fuliginosus:176 (Macaca_mulatta:25 'homo  sapiens':25):151):148);",
			nodes: []node{
				{id: 0, parent: -1, age: 324_000_000, children: []int{1, 2}},
				{id: 1, parent: 0, age: 0, taxon: "Gallus gallus", toRoot: 324_000_000, depth: 1},
				{id: 2, parent: 0, age: 176_000_000, children: []int{3, 4}, toRoot: 148_000_000, depth: 1},
				{id: 3, parent: 2, age: 0, taxon: "Macropus fuliginosus", toRoot: 324_000_000, depth: 2},
				{id: 4, parent: 2, age: 25_000_000, children: []int{5, 6}, toRoot: 299_000_000, depth: 2},
				{id: 5, parent: 4, age: 0, taxon: "Homo sapiens", toRoot: 324_000_000, depth: 3},
				{id: 6, parent: 4, age: 0, taxon: "Macaca mulatta", toRoot: 324_000_000, depth: 3},
			},
			terms: []string{
				"Gallus gallus",
				"Homo sapiens",
				"Macaca mulatta",
				"Macropus fuliginosus",
			},
			taxa: []string{
				"Gallus gallus",
				"Homo sapiens",
				"Macaca mulatta",
				"Macropus fuliginosus",
			},
			totLen: 849_000_000,
		},
		"non ultrametric": {
			name: "non ultrametric",
			in:   "(Eoraptor_lunensis:5, ((Ceratosaurus_nasicornis:25 'Carnotaurus sastrei':99):60,(Tyrannosaurus_rex:102,(Archaeopteryx_lithographica:10 Passer_domesticus:160):10):60):5);",
			nodes: []node{
				{id: 0, parent: -1, age: 235_000_000, children: []int{1, 2}},
				{id: 1, parent: 0, age: 230_000_000, taxon: "Eoraptor lunensis", toRoot: 5_000_000, depth: 1},
				{id: 2, parent: 0, age: 230_000_000, children: []int{3, 6}, toRoot: 5_000_000, depth: 1},
				{id: 3, parent: 2, age: 170_000_000, children: []int{4, 5}, toRoot: 65_000_000, depth: 2},
				{id: 4, parent: 3, age: 145_000_000, taxon: "Ceratosaurus nasicornis", toRoot: 90_000_000, depth: 3},
				{id: 5, parent: 3, age: 71_000_000, taxon: "Carnotaurus sastrei", toRoot: 164_000_000, depth: 3},
				{id: 6, parent: 2, age: 170_000_000, children: []int{7, 8}, toRoot: 65_000_000, depth: 2},
				{id: 7, parent: 6, age: 68_000_000, taxon: "Tyrannosaurus rex", toRoot: 167_000_000, depth: 3},
				{id: 8, parent: 6, age: 160_000_000, children: []int{9, 10}, toRoot: 75_000_000, depth: 3},
				{id: 9, parent: 8, age: 150_000_000, taxon: "Archaeopteryx lithographica", toRoot: 85_000_000, depth: 4},
				{id: 10, parent: 8, age: 0, taxon: "Passer domesticus", toRoot: 235_000_000, depth: 4},
			},
			terms: []string{
				"Archaeopteryx lithographica",
				"Carnotaurus sastrei",
				"Ceratosaurus nasicornis",
				"Eoraptor lunensis",
				"Passer domesticus",
				"Tyrannosaurus rex",
			},
			taxa: []string{
				"Archaeopteryx lithographica",
				"Carnotaurus sastrei",
				"Ceratosaurus nasicornis",
				"Eoraptor lunensis",
				"Passer domesticus",
				"Tyrannosaurus rex",
			},
			totLen: 536_000_000,
		},
		"no present taxon": {
			name: "no present taxon",
			in:   "(Eoraptor_lunensis:5, ((Ceratosaurus_nasicornis:25 'Carnotaurus sastrei':99):60,(Tyrannosaurus_rex:102, Archaeopteryx_lithographica:20):60):5);",
			age:  235_000_000,
			nodes: []node{
				{id: 0, parent: -1, age: 235_000_000, children: []int{1, 2}},
				{id: 1, parent: 0, age: 230_000_000, taxon: "Eoraptor lunensis", toRoot: 5_000_000, depth: 1},
				{id: 2, parent: 0, age: 230_000_000, children: []int{3, 6}, toRoot: 5_000_000, depth: 1},
				{id: 3, parent: 2, age: 170_000_000, children: []int{4, 5}, toRoot: 65_000_000, depth: 2},
				{id: 4, parent: 3, age: 150_000_000, taxon: "Archaeopteryx lithographica", toRoot: 85_000_000, depth: 3},
				{id: 5, parent: 3, age: 68_000_000, taxon: "Tyrannosaurus rex", toRoot: 167_000_000, depth: 3},
				{id: 6, parent: 2, age: 170_000_000, children: []int{7, 8}, toRoot: 65_000_000, depth: 2},
				{id: 7, parent: 6, age: 145_000_000, taxon: "Ceratosaurus nasicornis", toRoot: 90_000_000, depth: 3},
				{id: 8, parent: 6, age: 71_000_000, taxon: "Carnotaurus sastrei", toRoot: 164_000_000, depth: 3},
			},
			terms: []string{
				"Archaeopteryx lithographica",
				"Carnotaurus sastrei",
				"Ceratosaurus nasicornis",
				"Eoraptor lunensis",
				"Tyrannosaurus rex",
			},
			taxa: []string{
				"Archaeopteryx lithographica",
				"Carnotaurus sastrei",
				"Ceratosaurus nasicornis",
				"Eoraptor lunensis",
				"Tyrannosaurus rex",
			},
			totLen: 376_000_000,
		},
		"zero length branch": {
			name: "zero length branch",
			in:   "(A:10 (B:2, C:2):1e-25);",
			nodes: []node{
				{id: 0, parent: -1, age: 10_000_000, children: []int{1, 2}},
				{id: 1, parent: 0, age: 0, taxon: "A", toRoot: 10_000_000, depth: 1},
				{id: 2, parent: 0, age: 9_999_999, children: []int{3, 4}, toRoot: 1, depth: 1},
				{id: 3, parent: 2, age: 7_999_999, taxon: "B", toRoot: 2_000_001, depth: 2},
				{id: 4, parent: 2, age: 7_999_999, taxon: "C", toRoot: 2_000_001, depth: 2},
			},
			terms:  []string{"A", "B", "C"},
			taxa:   []string{"A", "B", "C"},
			totLen: 14_000_001,
		},
		"mesquite tree": {
			name: "mesquite tree",
			in:   "((A:1.0,B:1.0)298:2.4,C:3.4);",
			nodes: []node{
				{id: 0, parent: -1, age: 3_400_000, children: []int{1, 2}},
				{id: 1, parent: 0, taxon: "C", toRoot: 3_400_000, depth: 1},
				{id: 2, parent: 0, age: 1_000_000, children: []int{3, 4}, toRoot: 2_400_000, depth: 1},
				{id: 3, parent: 2, taxon: "A", toRoot: 3_400_000, depth: 2},
				{id: 4, parent: 2, taxon: "B", toRoot: 3_400_000, depth: 2},
			},
			terms:  []string{"A", "B", "C"},
			taxa:   []string{"A", "B", "C"},
			totLen: 7_800_000,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			coll, err := timetree.Newick(strings.NewReader(test.in), name, test.age)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", name, err)
			}
			names := coll.Names()
			if len(names) != 1 {
				t.Fatalf("%s: read %d trees, want %d", name, len(names), 1)
			}
			testTree(t, coll.Tree(names[0]), test)
		})
	}
}

func TestCollection(t *testing.T) {
	in := `
(Gallus_gallus:324,(Macropus_fuliginosus:176 (Macaca_mulatta:25 'homo  sapiens':25):151):148);
(Eoraptor_lunensis:5, ((Ceratosaurus_nasicornis:25 'Carnotaurus sastrei':99):60,(Tyrannosaurus_rex:102,(Archaeopteryx_lithographica:10 Passer_domesticus:160):10):60):5);
	`

	coll, err := timetree.Newick(strings.NewReader(in), "multiple", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := coll.Names()
	want := []string{"multiple", "multiple.1"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("tree names %v, want %v", names, want)
	}

	tests := []treeTest{
		{
			name: "multiple",
			in:   "(Gallus_gallus:324,(Macropus_fuliginosus:176 (Macaca_mulatta:25 'homo  sapiens':25):151):148);",
			nodes: []node{
				{id: 0, parent: -1, age: 324_000_000, children: []int{1, 2}},
				{id: 1, parent: 0, age: 0, taxon: "Gallus gallus", toRoot: 324_000_000, depth: 1},
				{id: 2, parent: 0, age: 176_000_000, children: []int{3, 4}, toRoot: 148_000_000, depth: 1},
				{id: 3, parent: 2, age: 0, taxon: "Macropus fuliginosus", toRoot: 324_000_000, depth: 2},
				{id: 4, parent: 2, age: 25_000_000, children: []int{5, 6}, toRoot: 299_000_000, depth: 2},
				{id: 5, parent: 4, age: 0, taxon: "Homo sapiens", toRoot: 324_000_000, depth: 3},
				{id: 6, parent: 4, age: 0, taxon: "Macaca mulatta", toRoot: 324_000_000, depth: 3},
			},
			terms: []string{
				"Gallus gallus",
				"Homo sapiens",
				"Macaca mulatta",
				"Macropus fuliginosus",
			},
			taxa: []string{
				"Gallus gallus",
				"Homo sapiens",
				"Macaca mulatta",
				"Macropus fuliginosus",
			},
			totLen: 849_000_000,
		},
		{
			name: "multiple.1",
			in:   "(Eoraptor_lunensis:5, ((Ceratosaurus_nasicornis:25 'Carnotaurus sastrei':99):60,(Tyrannosaurus_rex:102,(Archaeopteryx_lithographica:10 Passer_domesticus:160):10):60):5);",
			nodes: []node{
				{id: 0, parent: -1, age: 235_000_000, children: []int{1, 2}},
				{id: 1, parent: 0, age: 230_000_000, taxon: "Eoraptor lunensis", toRoot: 5_000_000, depth: 1},
				{id: 2, parent: 0, age: 230_000_000, children: []int{3, 6}, toRoot: 5_000_000, depth: 1},
				{id: 3, parent: 2, age: 170_000_000, children: []int{4, 5}, toRoot: 65_000_000, depth: 2},
				{id: 4, parent: 3, age: 145_000_000, taxon: "Ceratosaurus nasicornis", toRoot: 90_000_000, depth: 3},
				{id: 5, parent: 3, age: 71_000_000, taxon: "Carnotaurus sastrei", toRoot: 164_000_000, depth: 3},
				{id: 6, parent: 2, age: 170_000_000, children: []int{7, 8}, toRoot: 65_000_000, depth: 2},
				{id: 7, parent: 6, age: 68_000_000, taxon: "Tyrannosaurus rex", toRoot: 167_000_000, depth: 3},
				{id: 8, parent: 6, age: 160_000_000, children: []int{9, 10}, toRoot: 75_000_000, depth: 3},
				{id: 9, parent: 8, age: 150_000_000, taxon: "Archaeopteryx lithographica", toRoot: 85_000_000, depth: 4},
				{id: 10, parent: 8, age: 0, taxon: "Passer domesticus", toRoot: 235_000_000, depth: 4},
			},
			terms: []string{
				"Archaeopteryx lithographica",
				"Carnotaurus sastrei",
				"Ceratosaurus nasicornis",
				"Eoraptor lunensis",
				"Passer domesticus",
				"Tyrannosaurus rex",
			},
			taxa: []string{
				"Archaeopteryx lithographica",
				"Carnotaurus sastrei",
				"Ceratosaurus nasicornis",
				"Eoraptor lunensis",
				"Passer domesticus",
				"Tyrannosaurus rex",
			},
			totLen: 536_000_000,
		},
	}

	for _, test := range tests {
		testTree(t, coll.Tree(test.name), test)
	}
}

func TestNewickError(t *testing.T) {
	tests := map[string]struct {
		in  string
		age int64
		err error
	}{
		"unnamed tree": {
			err: timetree.ErrTreeNoName,
		},
		"not a tree": {
			in:  "not tree in the text",
			err: timetree.ErrNotNewick,
		},
		"unbalanced": {
			in:  "(((A:1,B);",
			err: timetree.ErrValSingleChild,
		},
		"empty node": {
			in:  "((),(C,D));",
			err: timetree.ErrValSingleChild,
		},
		"empty terminal": {
			in:  "(A,);",
			err: timetree.ErrValSingleChild,
		},
		"empty terminal (underlines)": {
			in:  "(A,____);",
			err: timetree.ErrValUnnamedTerm,
		},
		"empty terminal (spaces)": {
			in:  `(A, '  ');`,
			err: timetree.ErrValUnnamedTerm,
		},
		"unexpected branch length": {
			in:  "(A,(:1,C));",
			err: timetree.ErrUnexpBrLen,
		},
		"invalid branch length (terminal)": {
			in:  "(A:b, B:5);",
			err: timetree.ErrAddInvalidBrLen,
		},
		"invalid branch length (internal)": {
			in:  "(C:5, (A:5, B:5):x);",
			err: timetree.ErrAddInvalidBrLen,
		},
		"invalid age (terminal)": {
			in:  "(A:15, B:5);",
			age: 10_000_000,
			err: timetree.ErrInvalidRootAge,
		},
		"invalid age (internal)": {
			in:  "((A:5, B:5):10, C:5);",
			age: 10_000_000,
			err: timetree.ErrInvalidRootAge,
		},
		"repeated terminal": {
			in:  "(A, (A, B));",
			err: timetree.ErrAddRepeated,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			nm := "bad tree"
			if name == "unnamed tree" {
				nm = ""
			}

			_, err := timetree.Newick(strings.NewReader(test.in), nm, test.age)
			if err == nil {
				t.Fatalf("%s: invalid data %q: expecting error %q", name, test.in, test.err)
			}
			if !errors.Is(err, test.err) {
				t.Errorf("%s: invalid data %q: got error %q, want %q", name, test.in, err, test.err)
			}
		})
	}
}
