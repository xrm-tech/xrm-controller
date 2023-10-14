package ayaml

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

type item struct {
	Key       string
	Value     string
	ListStart bool
	Commented bool
	Comment   bool

	Indent int
	Line   uint
}

type parser struct {
	// indent    int
	// curIndent int
	line uint

	b       *bufio.Reader
	pending string
}

func newParser(r io.Reader) (p *parser, err error) {
	b := bufio.NewReader(r)
	var (
		pending string
		line    uint = 1
	)
	pending, err = readLine(b)
	if err != nil {
		return
	}
	if pending == "---\n" {
		// header, not required
		pending = ""
		line++
	}

	p = &parser{
		// indent:  -1,
		b:       b,
		pending: pending,
		line:    line,
	}

	return
}

func readLine(b *bufio.Reader) (line string, err error) {
	if line, err = b.ReadString('\n'); err == nil {
		if line == "\n" {
			line = ""
		} else {
			line = strings.TrimRight(line, "\n")
		}
	}
	return
}

var (
	reComment = regexp.MustCompile("^ *#.*")
)

// simple ansible yaml parser (not complete 2-level yaml with possible commented values)
func (p *parser) parse() (items []*item, err error) {
	// var newNode *Node
	for {
		if p.pending == "" {
			p.pending, err = readLine(p.b)
			if err != nil {
				if err == io.EOF {
					if len(items) > 0 {
						// cleanup blank lines at the end
						last := len(items) - 1
						if items[last].Comment && items[last].Key == "" {
							items = items[:last]
						}
					}
				}
				p.pending = ""
				return
			}
			p.line++
		}

		if p.pending == "" || reComment.MatchString(p.pending) {
			items = append(items, &item{
				Key:     p.pending,
				Comment: true,
				Line:    p.line,
			})
		} else {
			k, v, ok, commented := split(p.pending)
			if !ok {
				p.pending = ""
				continue
			}

			var (
				indent    int
				listStart bool
			)

			k, indent, listStart, err = splitIndent(k, p.line)
			if err != nil {
				p.pending = ""
				return
			}

			items = append(items, &item{
				Key:       k,
				Value:     v,
				ListStart: listStart,
				Commented: commented,
				Indent:    indent,
				Line:      p.line,
			})
		}

		p.pending = ""

	}
}
