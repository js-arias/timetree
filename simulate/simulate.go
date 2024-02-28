// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package simulate creates random trees.
package simulate

import (
	"fmt"
	"math/rand/v2"

	"github.com/js-arias/timetree"
)

// Rander is a distribution that returns
// a random number.
type Rander interface {
	Rand() float64
}

// Uniform creates a random tree using a uniform prior
// based on the method described by
// Ronquist et al. (2012)
// "A total evidence approach to dating with fossils,
// applied to the early radiation of Hymenoptera"
// Syst. Biol. 61: 973-999.
// doi:10.1093/sysbio/sys058.
// Uniform panics if len(ages) < 2,
func Uniform(name string, rootAge int64, ages []int64) *timetree.Tree {
	if len(ages) < 1 {
		panic("expecting more than two terminals")
	}

	for _, a := range ages {
		if a > rootAge {
			rootAge = a + 2
		}
	}

	// shuffle terminals
	rand.Shuffle(len(ages), func(i, j int) {
		ages[i], ages[j] = ages[j], ages[i]
	})

	added := make([]string, 0, len(ages))
	t := timetree.New(name, rootAge)
	// first node
	term := "term0"
	t.Add(0, rootAge-ages[0], term)
	added = append(added, term)
	term = "term1"
	t.Add(0, rootAge-ages[1], term)
	added = append(added, term)

	for i, a := range ages[2:] {
		// pick sister
		s := added[rand.IntN(i+2)]
		sis, _ := t.TaxNode(s)

		// pick age
		oldest := a
		if sa := t.Age(sis); sa > a {
			oldest = sa
		}
		age := rootAge - rand.Int64N(rootAge-oldest) + 1

		// search coalescent sister
		for {
			p := t.Parent(sis)
			pa := t.Age(p)
			if pa > age {
				break
			}
			sis = p
		}

		term := fmt.Sprintf("term%d", i+2)
		if _, err := t.AddSister(sis, a, age-a, term); err != nil {
			panic(fmt.Sprintf("unexpected error: %v", err))
		}
		added = append(added, term)
	}

	return t
}
