// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// Package format implements a command to format the trees
// in a tree file.
package format

import (
	"fmt"
	"io"
	"os"

	"github.com/js-arias/command"
	"github.com/js-arias/timetree"
)

var Command = &command.Command{
	Usage: "format [-o|--output <file>] [<treefile>...]",
	Short: "format trees in a file",
	Long: `
Command format reads one or more trees in TSV format, and formatted it by
sorting its node IDs.

One or more tree files can be given as arguments. If no file is given, it will
read the trees from the standard input.

The resulting tree file will be printed in the standard output. Use the flag
--output, or -o, to define an output file.
	`,
	SetFlags: setFlags,
	Run:      run,
}

var output string

func setFlags(c *command.Command) {
	c.Flags().StringVar(&output, "output", "", "")
	c.Flags().StringVar(&output, "o", "", "")
}

func run(c *command.Command, args []string) error {
	coll := timetree.NewCollection()

	if len(args) == 0 {
		args = append(args, "-")
	}
	for _, a := range args {
		nc, err := readCollection(c.Stdin(), a)
		if err != nil {
			return err
		}

		for _, tn := range nc.Names() {
			t := nc.Tree(tn)
			if err := coll.Add(t); err != nil {
				return fmt.Errorf("when adding trees from %q: %v", a, err)
			}
		}
	}

	ls := coll.Names()
	for _, tn := range ls {
		t := coll.Tree(tn)
		t.Format()
	}

	if err := writeTrees(c.Stdout(), coll); err != nil {
		return err
	}
	return nil
}

func readCollection(r io.Reader, name string) (*timetree.Collection, error) {
	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else {
		name = "stdin"
	}

	c, err := timetree.ReadTSV(r)
	if err != nil {
		return nil, fmt.Errorf("while reading file %q: %v", name, err)
	}
	return c, nil
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
