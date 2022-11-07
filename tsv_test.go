// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/js-arias/timetree"
)

func TestTSV(t *testing.T) {
	in := `
	(Eoraptor_lunensis:5, ((Ceratosaurus_nasicornis:25 'Carnotaurus sastrei':99):60,(Tyrannosaurus_rex:102,(Archaeopteryx_lithographica:10 Passer_domesticus:160):10):60):5);
	(Eoraptor_lunensis:5, ((Ceratosaurus_nasicornis:20 'Carnotaurus sastrei':94):65,(Tyrannosaurus_rex:102,(Archaeopteryx_lithographica:5 Passer_domesticus:155):15):60):5);
	`

	c, err := timetree.Newick(strings.NewReader(in), "dinosaurs", 0)
	if err != nil {
		t.Fatalf("while processing newick tree: %v", err)
	}

	var buf bytes.Buffer
	if err := c.TSV(&buf); err != nil {
		t.Fatalf("while writing data: %v", err)
	}

	nc, err := timetree.ReadTSV(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("while reading data: %v", err)
	}

	names := c.Names()
	if got := nc.Names(); !reflect.DeepEqual(got, names) {
		t.Errorf("read trees %v, want %v", got, names)
	}

	for _, name := range names {
		tr := c.Tree(name)
		nt := nc.Tree(name)
		if nt.Name() != tr.Name() {
			t.Errorf("tree name: got %q, want %q", nt.Name(), tr.Name())
		}

		for _, id := range tr.Nodes() {
			got := getNode(nt, id)
			want := getNode(tr, id)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("tree %s node %d: got %v, want %v", name, id, got, want)
			}

			if want.taxon == "" {
				continue
			}
			term, ok := nt.TaxNode(want.taxon)
			if !ok {
				t.Errorf("tree %s taxon %q: not found", name, want.taxon)
				continue
			}
			if term != id {
				t.Errorf("tree %s taxon %q: got ID %d, want %d\n", name, want.taxon, term, id)
			}
		}
	}
}
