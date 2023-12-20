package ayaml

import (
	"io"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

type NodeType uint8

const (
	NodeNil NodeType = iota
	NodeString
	NodeList
	NodeDict
	NodeLine
)

var mapNodeType = map[NodeType]string{
	NodeString: "string",
	NodeList:   "list",
	NodeDict:   "dict",
	NodeLine:   "line",
}

func (n NodeType) String() string {
	return mapNodeType[n]
}

type Node struct {
	Type      NodeType
	Key       string
	Value     interface{}
	Commented bool
	Deleted   bool

	Index  int
	Indent int
	Line   uint
	Parent *Node
}

func (node *Node) ParentKey() string {
	var keys []string
	p := node
	for p != nil {
		if p.Key == "" {
			if p.Index >= 0 {
				keys = append(keys, "["+strconv.Itoa(p.Index)+"]")
			} else {
				keys = append(keys, "[]")
			}
		} else {
			keys = append(keys, p.Key)
		}
		p = p.Parent
	}
	utils.StringsReverse(keys)

	return strings.Join(keys, ".")
}

func (node *Node) Locate(key string) (dict bool, n int) {
	if node.Type == NodeDict {
		dict = true
		if values, ok := node.Value.([]*Node); ok {
			for ; n < len(values); n++ {
				if values[n].Key == key {
					break
				}
			}
			if n == len(values) {
				n = -1
			}
		} else {
			n = -1
		}
	} else {
		n = -1
	}
	return
}

func (node *Node) deleteItem(i int) (ok bool) {
	values := node.Value.([]*Node)
	if i >= 0 && i < len(values) {
		node.Value.([]*Node)[i].Deleted = true
		ok = true
	}
	return
}

func (node *Node) Delete(key string) (ok bool) {
	if key == "" {
		return
	}
	_, i := node.Locate(key)
	if i != -1 {
		ok = node.deleteItem(i)
	}
	return
}

func (node *Node) DeleteItem(i int) (ok bool) {
	if i < 0 || node.Type != NodeList {
		return
	}
	return node.deleteItem(i)
}

func (node *Node) Set(key, value string) (ok bool) {
	if key == "" {
		return
	}
	var i int
	if ok, i = node.Locate(key); ok {
		values := node.Value.([]*Node)
		if i == -1 {
			node.Value = append(values, &Node{Type: NodeString, Value: value, Parent: node})
		} else {
			values[i].Type = NodeString
			values[i].Value = value
		}
	}
	return
}

func (node *Node) SetItem(i int, value string) (ok bool) {
	if i < -1 || node.Type != NodeList {
		return
	}
	values := node.Value.([]*Node)
	if i == -1 {
		node.Value = append(values, &Node{Type: NodeString, Value: value, Parent: node})
		ok = true
	} else if i < len(values) {
		values[i].Type = NodeString
		values[i].Value = value
		ok = true
	}
	return
}

func Decode(r io.Reader) (node *Node, err error) {
	p, err := newParser(r)
	if err == nil {
		var items []*item
		if items, err = p.parse(); err == nil {
			err = ErrorUnclosedYAML
			return
		}
		node, err = build(items)
	}

	return
}

func nextUncommented(items []*item, start int) (loc, comment int) {
	comment = -1
	loc = start
	for i := start; i < len(items); i++ {
		if items[i].Comment {
			if comment == -1 {
				comment = i
			}
		} else {
			loc = i
			return
		}
	}
	return
}

func hasUncommented(values []*Node) bool {
	for _, v := range values {
		if v.Type != NodeLine {
			return true
		}
	}
	return false
}

func buildComment(node *Node, item *item) {
	newNode := &Node{
		Type: NodeLine,
		Key:  item.Key,
		Line: item.Line,
	}
	if node.Value == nil {
		node.Value = []*Node{newNode}
	} else {
		node.Value = append(node.Value.([]*Node), newNode)
	}
}

func buildListItems(node *Node, items []*item) (i int, err error) {
	defer func() {
		if r := recover(); r != nil {
			var line uint
			if len(items) == 0 {
				line = 0
			} else {
				if i >= len(items) {
					i = len(items) - 1
				}
				line = items[i].Line
			}
			err = RecoverToError(r, line, string(debug.Stack()))
		}
	}()

	if node.Type != NodeList {
		err = NewParseError(items[0].Key, "list not init", items[0].Line)
		return
	}
	if len(items) == 0 {
		return
	}

	var (
		n       int
		newNode *Node
	)
	indent := items[0].Indent
	firstComment := items[0].Comment
	for len(items) > 0 {
		if items[0].Comment {
			buildComment(node, items[0])
			items = items[1:]
			i++
			continue
		} else if firstComment {
			indent = items[0].Indent
			firstComment = false
		}

		if items[0].Indent < indent {
			// back to down level
			break
		}
		if items[0].Indent != indent {
			err = NewIndentError(items[0].Key, "unindented list", items[0].Indent, items[0].Line)
			return
		}
		if !items[0].ListStart {
			err = NewParseError(items[0].Key, "list not started", items[0].Line)
			return
		}

		next, comment := nextUncommented(items, 1)
		_ = comment

		if len(items) == 1 || items[next].ListStart || items[next].Indent < indent {
			// simple string node
			newNode = &Node{
				Type: NodeString, Key: items[0].Key, Value: items[0].Value, Parent: node,
				Indent: items[0].Indent, Commented: items[0].Commented, Line: items[0].Line,
			}
			if node.Value == nil {
				node.Value = []*Node{newNode}
			} else {
				values := node.Value.([]*Node)
				newNode.Index = len(values)
				node.Value = append(values, newNode)
			}
			i++
			items = items[1:]
		} else if items[next].Indent < indent {
			// rewind
			break
		} else {
			// complex node
			newNode = &Node{
				Type: NodeDict, Parent: node,
				Indent: items[0].Indent, Commented: items[0].Commented,
			}
			if node.Value == nil {
				node.Value = []*Node{newNode}
			} else {
				values := node.Value.([]*Node)
				newNode.Index = len(values)
				node.Value = append(values, newNode)
			}

			n, err = buildDictItems(newNode, items)
			if err != nil {
				return
			}
			i += n
			items = items[n:]
		}

		if len(items) > 0 {
			if comment != -1 && next > comment && !items[0].ListStart {
				// may be next dict
				break
			} else {
				next, _ = nextUncommented(items, 0)
				if items[next].Indent < indent {
					// back to down level
					break
				}
			}
		}
	}

	return
}

func buildDictItems(node *Node, items []*item) (i int, err error) {
	defer func() {
		if r := recover(); r != nil {
			var line uint
			if len(items) == 0 {
				line = 0
			} else {
				if i >= len(items) {
					i = len(items) - 1
				}
				line = items[i].Line
			}
			err = RecoverToError(r, line, string(debug.Stack()))
		}
	}()

	if node.Type != NodeDict {
		err = NewParseError(items[0].Key, "dict init", items[0].Line)
		return
	}
	if node.Value != nil {
		if hasUncommented(node.Value.([]*Node)) {
			err = NewParseError(items[0].Key, "dict can't start with value", items[0].Line)
			return
		}
	}

	if len(items) == 0 {
		return
	}

	var (
		n       int
		newNode *Node
		lists   uint
	)

	for len(items) > 0 {
		if items[0].Comment {
			buildComment(node, items[0])
			items = items[1:]
			i++
			continue
		}

		if items[0].Indent < node.Indent {
			// back to down level
			return
		}
		if items[0].Indent > node.Indent {
			err = NewIndentError(items[0].Key, "unindented dict", items[0].Indent, items[0].Line)
			return
		}

		if items[0].ListStart {
			if lists > 0 {
				return
			}
			lists++
		}

		next, _ := nextUncommented(items, 1)

		if len(items) == 1 || (items[0].Indent >= items[next].Indent) {
			if items[0].Value == "" {
				newNode = &Node{
					Type: NodeDict, Key: items[0].Key, Parent: node,
					Indent: items[0].Indent, Commented: items[0].Commented,
				}
			} else {
				// simple string node
				newNode = &Node{
					Type: NodeString, Key: items[0].Key, Value: items[0].Value, Parent: node,
					Indent: items[0].Indent, Commented: items[0].Commented,
				}
			}

			if node.Value == nil {
				node.Value = []*Node{newNode}
			} else {
				values := node.Value.([]*Node)
				newNode.Index = len(values)
				node.Value = append(values, newNode)
			}

			i++
			items = items[1:]
		} else if items[next].Indent > items[0].Indent {
			if items[0].Value != "" {
				err = NewIndentError(items[0].Key, "complex list can not contain value", items[0].Indent, items[0].Line)
				return
			}
			// complex node
			if items[next].ListStart {
				newNode = &Node{
					Type: NodeList, Key: items[0].Key, Parent: node,
					Indent: items[0].Indent, Commented: items[0].Commented,
				}
				if node.Value == nil {
					node.Value = []*Node{newNode}
				} else {
					values := node.Value.([]*Node)
					newNode.Index = len(values)
					node.Value = append(values, newNode)
				}

				n, err = buildListItems(newNode, items[1:])
				if err != nil {
					return
				}
				next = n + 1
				i += next
				items = items[next:]
			} else {
				newNode = &Node{
					Type: NodeDict, Key: items[0].Key, Parent: node,
					Indent: items[0].Indent, Commented: items[0].Commented,
				}
				if node.Value == nil {
					node.Value = []*Node{newNode}
				} else {
					values := node.Value.([]*Node)
					newNode.Index = len(values)
					node.Value = append(values, newNode)
				}

				n, err = buildDictItems(newNode, items[next:])
				if err != nil {
					return
				}
				next += n + 1
				i += next
				items = items[next:]
			}
		} else {
			// back to down level
			break
		}

		if len(items) > 0 {
			if items[0].ListStart {
				// may be next dict
				break
			} else if items[0].Comment {
				next, _ = nextUncommented(items, 0)
				if items[next].Indent < node.Indent {
					// back to down level
					break
				}
			}
		}
	}

	return
}

func build(items []*item) (node *Node, err error) {
	var n int

	node = &Node{Type: NodeDict}

	for i := 0; i < len(items); {
		if items[i].Comment {
			buildComment(node, items[i])
			i++
		} else {
			if n, err = buildDictItems(node, items[i:]); err != nil {
				return
			}
			i += n
		}
	}

	// repack(node)

	return
}

// // shift commented
// func repack(node *Node) {
// 	if nodes, ok := node.Value.([]*Node); ok {
// 		for _, node := range nodes {
// 			if node.Key == "dr_import_storages" {
// 				storages := node.Value.([]*Node)
// 				last := len(storages) - 1

// 				storage := storages[last]

// 				values := storage.Value.([]*Node)
// 				j := len(values) - 1

// 				_ = storage
// 				break
// 			}
// 		}
// 	}
// }
