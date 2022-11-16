// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

// TimeTree is a tool to manipulate time calibrated phylogenetic trees.
package main

import (
	"github.com/js-arias/command"
	"github.com/js-arias/timetree/cmd/timetree/importcmd"
)

var app = &command.Command{
	Usage: "timetree <command> [<argument>...]",
	Short: "a tool to manipulate time calibrated phylogenetic trees",
}

func init() {
	app.Add(importcmd.Command)
}

func main() {
	app.Main()
}
