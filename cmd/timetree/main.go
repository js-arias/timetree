// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// TimeTree is a tool to manipulate time calibrated phylogenetic trees.
package main

import (
	"github.com/js-arias/command"
	"github.com/js-arias/timetree/cmd/timetree/draw"
	"github.com/js-arias/timetree/cmd/timetree/importcmd"
	"github.com/js-arias/timetree/cmd/timetree/list"
	"github.com/js-arias/timetree/cmd/timetree/newick"
	"github.com/js-arias/timetree/cmd/timetree/set"
	"github.com/js-arias/timetree/cmd/timetree/terms"
)

var app = &command.Command{
	Usage: "timetree <command> [<argument>...]",
	Short: "a tool to manipulate time calibrated phylogenetic trees",
}

func init() {
	app.Add(draw.Command)
	app.Add(importcmd.Command)
	app.Add(list.Command)
	app.Add(newick.Command)
	app.Add(set.Command)
	app.Add(terms.Command)
}

func main() {
	app.Main()
}
