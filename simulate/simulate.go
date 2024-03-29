// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package simulate creates random trees.
package simulate

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/js-arias/timetree"
	"gonum.org/v1/gonum/stat/distuv"
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
func Uniform(name string, max, min int64, ages []int64) *timetree.Tree {
	if len(ages) < 2 {
		panic("expecting more than two terminals")
	}

	for _, a := range ages[1:] {
		if a > min {
			min = a
		}
	}
	rootAge := max
	if max > min {
		rootAge = rand.Int64N(max-min) + min
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

// Coalescent creates a random tree
// using the Kingman coalescence
// with a population size of n.
// see Felsenstein J. (2004)
// "Inferring Phylogenies", Sinauer, p.456.
// Coalescent panics if terms < 2.
func Coalescent(name string, n float64, max int64, terms int) *timetree.Tree {
	if terms < 2 {
		panic("expecting more than two terminals")
	}

	ages := make([]int64, terms-1)
	for i := range ages {
		rate := float64((i+2)*(i+1)) / (4 * n)
		exp := distuv.Exponential{
			Rate: rate,
		}
		a := int64(exp.Rand())
		for a > max {
			a = int64(exp.Rand())
		}
		ages[i] = a
	}
	slices.SortFunc(ages, func(a, b int64) int {
		return cmp.Compare(b, a)
	})

	added := make([]string, 0, terms)
	t := timetree.New(name, ages[0])
	// first node
	term := "term0"
	t.Add(0, ages[0], term)
	added = append(added, term)
	term = "term1"
	t.Add(0, ages[0], term)
	added = append(added, term)

	for i := 2; i < terms; i++ {
		// pick sister
		s := added[rand.IntN(i)]
		sis, _ := t.TaxNode(s)

		// pick age
		age := ages[i-1]

		// search coalescent sister
		for {
			p := t.Parent(sis)
			pa := t.Age(p)
			if pa > age {
				break
			}
			sis = p
		}

		term := fmt.Sprintf("term%d", i)
		if _, err := t.AddSister(sis, 0, age, term); err != nil {
			panic(fmt.Sprintf("unexpected error: %v", err))
		}
		added = append(added, term)
	}

	return t
}

// Yule creates a Yule tree with the given speciation rate,
// in million years,
// stopping when the number of terminals is reached
// or when all proposed speciation events are in the future.
// It returns false if less than two terminals are included.
// Yule panics if terms < 2.
func Yule(name string, spRate float64, rootAge int64, terms int) (*timetree.Tree, bool) {
	if terms < 2 {
		panic("expecting more than two terminals")
	}

	exp := distuv.Exponential{
		Rate: spRate,
	}

	t := timetree.New(name, rootAge)
	added := 0
	yuleNode(t, 0, terms-2, &added, exp)

	if len(t.Terms()) < 2 {
		return t, false
	}

	return t, true
}

func yuleNode(t *timetree.Tree, n, max int, added *int, exp distuv.Exponential) {
	age := t.Age(n)
	if t.NumInternal() >= max {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
		term = fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
		return
	}

	// left descendant
	next := age - int64(exp.Rand()*1_000_000)
	if next < 0 {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
	} else {
		left, _ := t.Add(n, age-next, "")
		yuleNode(t, left, max, added, exp)
	}

	// right descendant
	if t.NumInternal() >= max {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
		return
	}

	next = age - int64(exp.Rand()*1_000_000)
	if next < 0 {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
		return
	}
	left, _ := t.Add(n, age-next, "")
	yuleNode(t, left, max, added, exp)
}

// BirthDeath create a birth-death tree
// with the given speciation and extinction rate,
// in million years,
// stopping when the number of terminals is reached
// of when all proposed events are in the future.
// It returns false if less than two terminals are included.
// BirthDeath panics if terms < 2.
func BirthDeath(name string, spRate, extRate float64, rootAge int64, terms int) (*timetree.Tree, bool) {
	if terms < 2 {
		panic("expecting more than two terminals")
	}

	if extRate == 0 {
		return Yule(name, spRate, rootAge, terms)
	}

	sp := distuv.Exponential{
		Rate: spRate,
	}
	e := distuv.Exponential{
		Rate: extRate,
	}

	t := timetree.New(name, rootAge)
	added := 0
	bdNode(t, 0, terms-2, &added, sp, e)

	if len(t.Terms()) < 2 {
		return t, false
	}

	return t, true
}

func bdNode(t *timetree.Tree, n, max int, added *int, sp, ext distuv.Exponential) {
	age := t.Age(n)
	if t.NumInternal() >= max {
		// left descendant
		brLen := age
		if e := age - int64(ext.Rand()*1_000_000); e > 0 {
			brLen = age - e
		}
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, brLen, term)
		*added++

		// right descendant
		brLen = age
		if e := age - int64(ext.Rand()*1_000_000); e > 0 {
			brLen = age - e
		}
		term = fmt.Sprintf("term%d", *added)
		t.Add(n, brLen, term)
		*added++
		return
	}

	// left descendant
	spNext := age - int64(sp.Rand()*1_000_000)
	eNext := age - int64(ext.Rand()*1_000_000)
	if spNext < 0 && eNext < 0 {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
	} else if eNext > spNext {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age-eNext, term)
		*added++
	} else {
		left, _ := t.Add(n, age-spNext, "")
		bdNode(t, left, max, added, sp, ext)
	}

	// right descendant
	eNext = age - int64(ext.Rand()*1_000_000)
	if t.NumInternal() >= max {
		brLen := age
		if eNext > 0 {
			brLen = age - eNext
		}
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, brLen, term)
		*added++
		return
	}

	spNext = age - int64(sp.Rand()*1_000_000)
	if spNext < 0 && eNext < 0 {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age, term)
		*added++
		return
	}
	if eNext > spNext {
		term := fmt.Sprintf("term%d", *added)
		t.Add(n, age-eNext, term)
		*added++
		return
	}
	right, _ := t.Add(n, age-spNext, "")
	bdNode(t, right, max, added, sp, ext)
}
