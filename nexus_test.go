// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree_test

import (
	"strings"
	"testing"

	"github.com/js-arias/timetree"
)

var nexusTest = `#NEXUS

Begin taxa;
	Dimensions ntax=6;
	Taxlabels
		Eoraptor_lunensis
		Ceratosaurus_nasicornis
		'Carnotaurus sastrei'
		Tyrannosaurus_rex
		Archaeopteryx_lithographica
		Passer_domesticus
	;
End;

Begin trees;
	Translate
		1 Eoraptor_lunensis,
		2 Ceratosaurus_nasicornis,
		3 'Carnotaurus sastrei',
		4 Tyrannosaurus_rex,
		5 Archaeopteryx_lithographica,
		6 Passer_domesticus
		;
	tree * untitled = [&R](1:5,((2:25,3:99):60,(4:102,(5:10,6:160):10):60):5);
	tree untitled = [&R](1:5,((2:25,3:99):60,(4:102,(5:10,6:160):10):60):5):10.0;
End;
`

func TestNexus(t *testing.T) {
	want := treeTest{
		name: "untitled",
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
	}

	coll, err := timetree.Nexus(strings.NewReader(nexusTest), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := coll.Names()
	if len(names) != 2 {
		t.Fatalf("nexus: read %d trees, want %d", len(names), 1)
	}
	testTree(t, coll.Tree("untitled"), want)
	want.name = "untitled.1"
	testTree(t, coll.Tree("untitled.1"), want)
}
