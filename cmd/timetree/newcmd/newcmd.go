// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package newcmd implements a command to create a new bush tree
// from a list of taxa.
package newcmd

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: `new [--format <value>]
	[--age <value>]
	[-o|--output <file>]
	--name <tree-name>
	[<file>...]`,
	Short: "creates a tree from a taxa list",
	Long: `
Command new reads one or more files with taxon names and creates a new bush
tree.

One or more files can be used as argument of the command. If no file is given
the names will be read from the standard input.

The name of the tree should be defined with the flag --name.

By default, the names will be read one name per line, ignoring lines starting
with sharp symbol ('#'). The age of the names will be set as zero. If the flag
--format is used, different kinds of files will be used:

	text the default value
	tsv  a tab-delimited file, one column should be "species" for the
	     taxon names, and, if defined, a column "age" should be used for
	     the taxon (in million years).
	csv  the same as a tsv file, but using comma delimited fields.

By default, the age of the tree will be set to be 2 million years older than
the oldest defined taxon. Use the flag --age to define a particular age. Any
read taxon older than the defined root age, they will be ignored.

By default the resulting tree will be printed in the standard output. Uee the
flag --output, or -o, to define an output file. If the file already exist, the
tree will be added to the file.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var output string
var ageFlag float64
var nameFlag string
var format string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
	c.Flags().StringVar(&nameFlag, "name", "", "")
	c.Flags().StringVar(&format, "format", "text", "")
	c.Flags().Float64Var(&ageFlag, "age", 0, "")
}

func run(c *command.Command, args []string) error {
	nameFlag = strings.ToLower(strings.Join(strings.Fields(nameFlag), " "))
	if nameFlag == "" {
		return c.UsageError("undefined --name flag")
	}

	fn := readNamesText
	format = strings.ToLower(format)
	switch format {
	case "text":
	case "tsv":
		fn = readNamesTable
	case "csv":
		fn = readNamesTable
	default:
		return c.UsageError(fmt.Sprintf("unknown format %q", format))
	}

	coll, err := newTreeCollection()
	if err != nil {
		return err
	}

	tax := make(map[string]*taxonName)

	if len(args) == 0 {
		args = append(args, "-")
	}
	for _, a := range args {
		if err := fn(c.Stdin(), a, tax); err != nil {
			return err
		}
	}

	// creates the tree
	maxAge := int64(ageFlag * 1_000_000)
	if maxAge == 0 {
		for _, tx := range tax {
			if tx.age > maxAge {
				maxAge = tx.age
			}
		}
		maxAge += 2_000_000
	}
	tr := timetree.New(nameFlag, maxAge)
	for _, tx := range tax {
		if tx.age > maxAge {
			continue
		}
		if _, err := tr.Add(tr.Root(), maxAge-tx.age, tx.name); err != nil {
			return err
		}
	}

	// add the tree
	if len(tr.Taxa()) == 0 {
		return nil
	}
	if err := coll.Add(tr); err != nil {
		return err
	}
	if err := writeTrees(c.Stdout(), coll); err != nil {
		return err
	}
	return nil
}

func newTreeCollection() (*timetree.Collection, error) {
	if output == "" {
		return timetree.NewCollection(), nil
	}

	f, err := os.Open(output)
	if errors.Is(err, os.ErrNotExist) {
		return timetree.NewCollection(), nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c, err := timetree.ReadTSV(f)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", output, err)
	}
	return c, nil
}

type taxonName struct {
	name string
	age  int64
}

func readNamesTable(r io.Reader, input string, tax map[string]*taxonName) error {
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	} else {
		input = "stdin"
	}

	tab := csv.NewReader(r)
	if format == "tsv" {
		tab.Comma = '\t'
	}
	tab.Comment = '#'

	header, err := tab.Read()
	if err != nil {
		return fmt.Errorf("%q: header: %v", input, err)
	}
	fields := make(map[string]int)
	for i, h := range header {
		h = strings.ToLower(strings.Join(strings.Fields(h), " "))
		fields[h] = i
	}
	if _, ok := fields["species"]; !ok {
		return fmt.Errorf("%q: header: expecting %q field", input, "species")
	}

	for {
		row, err := tab.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		ln, _ := tab.FieldPos(0)
		if err != nil {
			return fmt.Errorf("%q: on row %d: %v", input, ln, err)
		}

		f := "species"
		name := strings.ToLower(strings.Join(strings.Fields(row[fields[f]]), " "))
		if name == "" {
			continue
		}

		var age int64
		f = "age"
		if pos, ok := fields[f]; ok {
			mya, err := strconv.ParseFloat(row[pos], 64)
			if err != nil {
				return fmt.Errorf("%q: on row %d: field %q: %v", input, ln, f, err)
			}
			age = int64(mya * 1_000_000)
		}
		tx, ok := tax[name]
		if !ok {
			tx = &taxonName{
				name: name,
				age:  age,
			}
			tax[name] = tx
		}
		if age > tx.age {
			tx.age = age
		}
	}
	return nil
}

func readNamesText(r io.Reader, input string, tax map[string]*taxonName) error {
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	} else {
		input = "stdin"
	}

	br := bufio.NewReader(r)
	for ln := 1; ; ln++ {
		line, err := br.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%q: on row %d: %v", input, ln, err)
		}

		line = strings.ToLower(strings.Join(strings.Fields(line), " "))
		if line == "" {
			continue
		}
		if line[0] == '#' {
			continue
		}
		tax[line] = &taxonName{
			name: line,
		}
	}
	return nil
}

func writeTrees(w io.Writer, c *timetree.Collection) (err error) {
	outName := "stdout"
	if output != "" {
		outName = output
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer func() {
			e := f.Close()
			if e != nil && err == nil {
				err = e
			}
		}()
		w = f
	}

	if err := c.TSV(w); err != nil {
		return fmt.Errorf("while writing to %q: %v", outName, err)
	}
	return nil
}
