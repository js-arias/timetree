// Copyright © 2022 J. Salvador Arias <jsalarias@gmail.com>
// All rights reserved.
// Distributed under BSD2 license that can be found in the LICENSE file.

package draw

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/js-arias/timetree"
)

const yStep = 12

type node struct {
	x    float64
	y    int
	topY int
	botY int

	id  int
	tax string
	age float64

	anc  *node
	desc []*node
}

type svgTree struct {
	y     int
	x     float64
	taxSz int
	root  *node
}

// millionYears is used to transform ages
// (an integer in years)
// to a float in million years.
const millionYears = 1_000_000

func copyTree(t *timetree.Tree, xStep float64) svgTree {
	maxSz := 0
	var root *node
	ids := make(map[int]*node)
	for _, id := range t.Nodes() {
		var anc *node
		p := t.Parent(id)
		if p >= 0 {
			anc = ids[p]
		}

		n := &node{
			id:  id,
			tax: t.Taxon(id),
			anc: anc,
			age: float64(t.Age(id)) / millionYears,
		}
		if anc == nil {
			root = n
		} else {
			anc.desc = append(anc.desc, n)
		}
		ids[id] = n
		if len(n.tax) > maxSz {
			maxSz = len(n.tax)
		}
	}

	s := svgTree{root: root}
	s.prepare(root, xStep)
	s.y = s.y * yStep
	s.taxSz = maxSz

	return s
}

func (s *svgTree) prepare(n *node, xStep float64) {
	n.x = (s.root.age-n.age)*xStep + 10
	if s.x < n.x {
		s.x = n.x
	}

	if n.desc == nil {
		n.y = s.y*yStep + 5
		s.y += 1
		return
	}

	botY := 0
	topY := math.MaxInt
	for _, d := range n.desc {
		s.prepare(d, xStep)
		if d.y < topY {
			topY = d.y
		}
		if d.y > botY {
			botY = d.y
		}
	}
	n.topY = topY
	n.botY = botY
	n.y = topY + (botY-topY)/2
}

func (s svgTree) draw(w io.Writer) error {
	fmt.Fprintf(w, "%s", xml.Header)
	e := xml.NewEncoder(w)
	svg := xml.StartElement{
		Name: xml.Name{Local: "svg"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "height"}, Value: strconv.Itoa(s.y + 5)},
			// assume that each character has 6 pixels wide
			{Name: xml.Name{Local: "width"}, Value: strconv.Itoa(int(s.x) + s.taxSz*6)},
			{Name: xml.Name{Local: "xmlns"}, Value: "http://www.w3.org/2000/svg"},
		},
	}
	e.EncodeToken(svg)

	g := xml.StartElement{
		Name: xml.Name{Local: "g"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "stroke-width"}, Value: "2"},
			{Name: xml.Name{Local: "stroke"}, Value: "black"},
			{Name: xml.Name{Local: "stroke-linecap"}, Value: "round"},
			{Name: xml.Name{Local: "font-family"}, Value: "Verdana"},
			{Name: xml.Name{Local: "font-size"}, Value: "10"},
		},
	}
	e.EncodeToken(g)

	s.root.draw(e)
	s.root.label(e)

	e.EncodeToken(g.End())
	e.EncodeToken(svg.End())
	if err := e.Flush(); err != nil {
		return err
	}
	return nil
}

func (n node) draw(e *xml.Encoder) {
	// horizontal line
	ln := xml.StartElement{
		Name: xml.Name{Local: "line"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "x1"}, Value: strconv.Itoa(int(n.x - 5))},
			{Name: xml.Name{Local: "y1"}, Value: strconv.Itoa(int(n.y))},
			{Name: xml.Name{Local: "x2"}, Value: strconv.Itoa(int(n.x))},
			{Name: xml.Name{Local: "y2"}, Value: strconv.Itoa(int(n.y))},
		},
	}
	if n.anc != nil {
		ln.Attr[0].Value = strconv.Itoa(int(n.anc.x))
	}
	e.EncodeToken(ln)
	e.EncodeToken(ln.End())

	// terminal name
	if n.desc == nil {
		return
	}

	// draws vertical line
	ln.Attr[0].Value = ln.Attr[2].Value
	ln.Attr[1].Value = strconv.Itoa(int(n.topY))
	ln.Attr[3].Value = strconv.Itoa(int(n.botY))
	e.EncodeToken(ln)
	e.EncodeToken(ln.End())

	for _, d := range n.desc {
		d.draw(e)
	}
}

func (n node) label(e *xml.Encoder) {
	if n.desc == nil {
		tx := xml.StartElement{
			Name: xml.Name{Local: "text"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "x"}, Value: strconv.Itoa(int(n.x + 10))},
				{Name: xml.Name{Local: "y"}, Value: strconv.Itoa(int(n.y + 5))},
				{Name: xml.Name{Local: "stroke-width"}, Value: "0"},
				{Name: xml.Name{Local: "font-style"}, Value: "italic"},
			},
		}
		e.EncodeToken(tx)
		e.EncodeToken(xml.CharData(n.tax))
		e.EncodeToken(tx.End())
	}

	// draws a circle at the node
	circ := xml.StartElement{
		Name: xml.Name{Local: "circle"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "cx"}, Value: strconv.Itoa(int(n.x))},
			{Name: xml.Name{Local: "cy"}, Value: strconv.Itoa(int(n.y))},
			{Name: xml.Name{Local: "r"}, Value: "7"},
			{Name: xml.Name{Local: "fill"}, Value: "white"},
			{Name: xml.Name{Local: "stroke"}, Value: "black"},
			{Name: xml.Name{Local: "stroke-width"}, Value: "1"},
		},
	}
	e.EncodeToken(circ)
	e.EncodeToken(circ.End())

	// put node ID
	tx := xml.StartElement{
		Name: xml.Name{Local: "text"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "x"}, Value: strconv.Itoa(int(n.x - 5))},
			{Name: xml.Name{Local: "y"}, Value: strconv.Itoa(int(n.y + 2))},
			{Name: xml.Name{Local: "stroke-width"}, Value: "0"},
			{Name: xml.Name{Local: "font-size"}, Value: "6"},
		},
	}
	e.EncodeToken(tx)
	e.EncodeToken(xml.CharData(strconv.Itoa(n.id)))
	e.EncodeToken(tx.End())

	for _, d := range n.desc {
		d.label(e)
	}
}
