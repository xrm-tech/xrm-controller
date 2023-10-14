package ayaml

import (
	"bufio"
	"errors"
)

func writeStringLn(w *bufio.Writer, s string) (err error) {
	if _, err = w.WriteString(s); err != nil {
		return
	}
	err = w.WriteByte('\n')
	return
}

func writeKVLn(w *bufio.Writer, k, v string) (err error) {
	if _, err = w.WriteString(k); err != nil {
		return
	}
	if _, err = w.WriteString(": "); err != nil {
		return
	}
	if _, err = w.WriteString(v); err != nil {
		return
	}
	err = w.WriteByte('\n')
	return
}

func writeIndent(bw *bufio.Writer, n int) (err error) {
	for i := 0; i < n; i++ {
		if err = bw.WriteByte(' '); err != nil {
			break
		}
	}
	return
}

func writeListIndent(bw *bufio.Writer, n int) (err error) {
	for i := 0; i < n-2; i++ {
		if err = bw.WriteByte(' '); err != nil {
			break
		}
	}
	if err = bw.WriteByte('-'); err == nil {
		err = bw.WriteByte(' ')
	}
	return
}

func (node *Node) write(w *bufio.Writer, indent bool) (err error) {
	switch node.Type {
	case NodeString:
		if indent {
			if err = writeIndent(w, node.Indent); err != nil {
				return
			}
		}
		err = writeKVLn(w, node.Key, node.Value.(string))
	case NodeDict:
		if node.Key != "" {
			if indent {
				if err = writeIndent(w, node.Indent); err != nil {
					return
				}
			}

			if _, err = w.WriteString(node.Key); err != nil {
				return
			}
			if _, err = w.WriteString(":\n"); err != nil {
				return
			}
			indent = true
		}
		if node.Value != nil {
			values := node.Value.([]*Node)
			for i, v := range values {
				if err = v.write(w, indent || i > 0); err != nil {
					break
				}
			}
		}
	case NodeList:
		if node.Key != "" {
			if indent {
				if err = writeIndent(w, node.Indent); err != nil {
					return
				}
			}

			if _, err = w.WriteString(node.Key); err != nil {
				return
			}
			if _, err = w.WriteString(":\n"); err != nil {
				return
			}
		}
		if node.Value != nil {
			values := node.Value.([]*Node)
			for _, v := range values {
				if err = writeListIndent(w, node.Indent); err != nil {
					return
				}
				if err = v.write(w, false); err != nil {
					break
				}
			}
		}
	case NodeLine:
		err = writeStringLn(w, node.Key)
	default:
		err = errors.New("unhandled node: " + node.Type.String())
	}

	return
}

func (node *Node) Write(w *bufio.Writer) (err error) {
	if _, err = w.WriteString("---\n"); err != nil {
		return
	}
	values := node.Value.([]*Node)
	// doubleEndl := len(values) > 1
	// l := len(values) - 1
	for _, v := range values {
		if err = v.write(w, true); err != nil {
			break
		}
		// if doubleEndl && i < l && values[i+1].Type != NodeString {
		// 	if _, err = w.WriteString("\n"); err != nil {
		// 		break
		// 	}
		// }
	}

	if err == nil {
		err = w.Flush()
	}

	return
}
