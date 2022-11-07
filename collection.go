// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package timetree

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

// Tree collection errors
var (
	ErrTreeNoName   = errors.New("tree without name")
	ErrTreeRepeated = errors.New("repeated tree name")
)

// A Collection is a collection of phylogenetic trees.
type Collection struct {
	trees map[string]*Tree
}

// NewCollection returns a new empty collection.
func NewCollection() *Collection {
	return &Collection{
		trees: make(map[string]*Tree),
	}
}

// Add adds a tree to a tree collection.
// It will return an error if a the collection
// has a tree with the name of the added tree
// or the tree name is empty.
func (c *Collection) Add(t *Tree) error {
	name := strings.ToLower(strings.Join(strings.Fields(t.Name()), " "))
	if name == "" {
		return ErrTreeNoName
	}
	if _, dup := c.trees[name]; dup {
		return fmt.Errorf("%w: %s", ErrTreeRepeated, name)
	}
	c.trees[name] = t
	return nil
}

// Names return the names of the trees in the collection.
func (c *Collection) Names() []string {
	names := make([]string, 0, len(c.trees))
	for _, t := range c.trees {
		names = append(names, t.name)
	}
	slices.Sort(names)
	return names
}

// Tree returns a tree with a given name.
func (c *Collection) Tree(name string) *Tree {
	name = strings.ToLower(strings.Join(strings.Fields(name), " "))
	if name == "" {
		return nil
	}
	return c.trees[name]
}
