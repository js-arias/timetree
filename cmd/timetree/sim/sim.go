// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package sim implements a command to simulate
// a phylogenetic tree.
package sim

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
	"github.com/js-arias/timetree/simulate"
)

var Command = &command.Command{
	Usage: `sim [-o|--output <file>] [--name <tree-name>]
	[--trees <tree-number]
	[--coalescent <number>]
	[--yule <rate>]
	[--bd <rate,rate>]
	--terms <term-number> [--min <age>] --max <age>`,
	Short: "simulate trees",
	Long: `
Command sim creates one on more random trees.

By default, the output will be printed in the standard output. Use the flag
--output, or -o, to define an output file. It will replace any previous file.

By default, the trees will be named "random-tree" with a number. Use the flag
--name to modify the prefix name of the tree.

By default, a single tree will be created. Use the flag --trees to define a
different number of trees.

The flag --terms is required and indicates the number of terms that the tree
should have.

The flags --min and --max define the minimum and maximum ages of the root
node in million years. The flag --max is required. The flag --min can be
omitted; its default value is 0.000001 (i.e. a year before present).

By default, it creates uniform trees. Use the flag --coalescent with the "size
of the population" to create a coalescent tree. A rule of thumb using as size
the same value of the maximum age. Use the flag --yule with the speciation rate
per million years to create a Yule tree. Use the flag --bd with an speciation
and extinction rate per million years to create a birth-death tree, the format
for the rates are "<value>,<value>" for example "0.1,0.01" will indicate a
speciation rate of 0.1 and an extinction rate of 0.01.

	`,
	SetFlags: setFlags,
	Run:      run,
}

var output string
var nameFlag string
var birthDeath string
var numTrees int
var numTerms int
var minAge float64
var maxAge float64
var coalescent float64
var yule float64

func setFlags(c *command.Command) {
	c.Flags().IntVar(&numTrees, "trees", 1, "")
	c.Flags().IntVar(&numTerms, "terms", 0, "")
	c.Flags().Float64Var(&maxAge, "max", 0, "")
	c.Flags().Float64Var(&minAge, "min", 0, "")
	c.Flags().Float64Var(&coalescent, "coalescent", 0, "")
	c.Flags().Float64Var(&yule, "yule", 0, "")
	c.Flags().StringVar(&birthDeath, "bd", "", "")
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
	c.Flags().StringVar(&nameFlag, "name", "random-tree", "")
}

const millionYears = 1_000_000

func run(c *command.Command, args []string) (err error) {
	if numTerms <= 0 {
		return c.UsageError("flag --terms must be defined")
	}
	if maxAge <= 0 {
		return c.UsageError("flag --max must be defined")
	}
	min := int64(minAge * millionYears)
	max := int64(maxAge * millionYears)
	if min > max {
		max = min
	}
	if min == 0 {
		min = 1
	}

	var spRate, extRate float64
	if birthDeath != "" {
		var err error
		spRate, extRate, err = parseRates()
		if err != nil {
			return err
		}
		if extRate == 0 && yule == 0 {
			yule = spRate
		}
	}

	ages := make([]int64, numTerms)

	coll := timetree.NewCollection()
	for i := 0; i < numTrees; i++ {
		name := fmt.Sprintf("%s-%d", nameFlag, i)

		var t *timetree.Tree
		switch {
		case extRate > 0:
			root := max
			if min < max {
				root = rand.Int64N(max-min) + min
			}
			for {
				var ok bool
				t, ok = simulate.BirthDeath(name, spRate, extRate, root, numTerms)
				if ok {
					break
				}
			}
		case yule > 0:
			root := max
			if min < max {
				root = rand.Int64N(max-min) + min
			}
			for {
				var ok bool
				t, ok = simulate.Yule(name, yule, root, numTerms)
				if ok {
					break
				}
			}
		case coalescent > 0:
			t = simulate.Coalescent(name, coalescent*millionYears, max, numTerms)
		default:
			t = simulate.Uniform(name, max, min, ages)
		}
		t.Format()
		coll.Add(t)
	}

	w := c.Stdout()
	if output != "" {
		var f *os.File
		f, err = os.Create(output)
		if err != nil {
			return err
		}
		w = f
		defer func() {
			e := f.Close()
			if e != nil && err == nil {
				err = e
			}
		}()
	} else {
		output = "stdout"
	}

	if err := coll.TSV(w); err != nil {
		return fmt.Errorf("while writing to %q: %v", output, err)
	}

	return nil
}

func parseRates() (sp, e float64, err error) {
	sv := strings.Split(birthDeath, ",")
	if len(sv) != 2 {
		return 0, 0, fmt.Errorf("flag --bd: expecting '<value>,<value>'")
	}

	sp, err = strconv.ParseFloat(sv[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("flag --bd: %v", err)
	}
	if sp < 0 {
		return 0, 0, fmt.Errorf("flag --bd: invalid speciation rate %.6f", sp)
	}

	e, err = strconv.ParseFloat(sv[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("flag --bd: %v", err)
	}
	if e < 0 {
		return 0, 0, fmt.Errorf("flag --bd: invalid extinction rate %.6f", e)
	}

	return sp, e, nil
}
