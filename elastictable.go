package elastictable

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

const defaultPadding = " "
const defaultBorder = "|"
const defaultDivider = "-"
const defaultDividerBorder = "+"
const defaultDividerPadding = "-"
const defaultMargin = 2*len(defaultPadding) + len(defaultBorder)

type elasticCol struct {
	index  int
	min    int
	max    int
	width  int
	height int
}

// ElasticTable - struct that is returned
type ElasticTable struct {
	cols   []elasticCol
	header []string
	rows   [][]string
}

// NewElasticTable - creates a new ElasticTable with given headers
func NewElasticTable(header []string) *ElasticTable {
	e := &ElasticTable{
		header: header,
		cols:   make([]elasticCol, len(header)),
		rows:   [][]string{},
	}

	for i, v := range header {
		l, c := runeWidth(v), &e.cols[i]
		c.index = i
		c.min, c.max, c.width = l, l, l
		c.height = 1
	}
	return e
}

// AddRow - Adds a row
func (e *ElasticTable) AddRow(row []string) {
	e.rows = append(e.rows, row)
	for i, v := range row {
		l, c := runeWidth(v), &e.cols[i]
		if l < c.min {
			c.min = l
		}
		if l > c.max {
			c.max, c.width = l, l
		}
	}
}

// Render - Prints formatted table to the given writer
func (e *ElasticTable) Render(out io.Writer) {
	widths := e.optimizedWidths()
	dividers := make([]string, len(widths))
	for i, v := range widths {
		dividers[i] = strings.Repeat(defaultDivider, v)
	}
	printRow(out, e.header, widths, defaultBorder, defaultPadding)
	printRow(out, dividers, widths, defaultDividerBorder, defaultDividerPadding)
	for _, row := range e.rows {
		printRow(out, row, widths, defaultBorder, defaultPadding)
	}
}

func (e *ElasticTable) mapWidths(f func(col elasticCol) int) []int {
	out := make([]int, len(e.cols))
	for _, v := range e.cols {
		out[v.index] = f(v)
	}
	return out
}

func (e *ElasticTable) optimizedWidths() []int {
	num := len(e.cols)
	termWidth := termWidth() - (num * defaultMargin)
	sort.Sort(elasticSortMax(e.cols))

	minTot, maxTot := 0, 0
	for _, v := range e.cols {
		minTot = minTot + v.min
		maxTot = maxTot + v.width
	}

	if minTot >= termWidth {
		return e.mapWidths(func(col elasticCol) int { return col.min })
	} else if maxTot <= termWidth {
		return e.mapWidths(func(col elasticCol) int { return col.max })
	}

OUTER:
	for {
		if maxTot <= termWidth {
			break
		}

		for i := 0; i < num-1; i++ {
			curr, next := &e.cols[i], &e.cols[i+1]
			width := int(math.Ceil(float64(curr.max) / float64(curr.height+1)))
			if width >= next.width {
				maxTot = maxTot - (curr.width - width)
				curr.height = curr.height + 1
				curr.width = width
				continue OUTER
			}
		}

		// no further optimizations can be performed
		break
	}

	return e.mapWidths(func(col elasticCol) int {
		// distribute remaining space (negative or positive ratio)
		return col.width + int(math.Floor(float64(termWidth-maxTot)*float64(col.width)/float64(maxTot)))
	})
}

type elasticSortMax []elasticCol

func (s elasticSortMax) Len() int {
	return len(s)
}
func (s elasticSortMax) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s elasticSortMax) Less(i int, j int) bool {
	return s[i].max >= s[j].max
}

func printRow(out io.Writer, row []string, widths []int, border string, padding string) {
	colmax := len(row)
	subrows := make([][]string, colmax)
	submax := 1
	for i, w := range widths {
		subrows[i] = wrapString(row[i], w)
		if len(subrows[i]) > submax {
			submax = len(subrows[i])
		}
	}

	for sub := 0; sub < submax; sub++ {
		for i, w := range widths {
			str := ""
			format := padding + "%-" + strconv.Itoa(w) + "s" + padding
			if i < colmax-1 {
				format = format + border
			} else {
				format = format + "\n"
			}
			if sub < len(subrows[i]) {
				str = subrows[i][sub]
			}
			fmt.Fprintf(out, format, str)
		}
	}
}
