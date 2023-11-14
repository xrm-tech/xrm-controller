package ayaml

import "strings"

type FilterType uint8

const (
	FilterNil FilterType = iota
	FilterKey
	FilterList
)

type Filter struct {
	Type FilterType
	Key  string
}

func newFilter(key string) Filter {
	key = strings.Trim(key, " ")
	if strings.HasSuffix(key, "[]") {
		key = strings.TrimRight(key[:len(key)-2], " ")
		return Filter{Type: FilterList, Key: key}
	}
	// TODO: delete item (list) with [N]

	return Filter{Type: FilterKey, Key: key}
}

func Remap(nodes *Node, remap map[string]string) {
	for k, v := range remap {
		keys := strings.Split(k, ".")
		if len(keys) == 0 || keys[0] == "" {
			continue
		}

		remapNodes(nodes, keys, v)
	}
}

func remapNodes(node *Node, keys []string, value string) {
	if len(keys) == 0 {
		return
	}

	if _, ok := node.Value.([]*Node); ok {
		if len(keys) == 1 {
			if value == "~" {
				node.Delete(keys[0])
			} else {
				node.Set(keys[0], value)
			}
		} else {
			filter := newFilter(keys[0])
			if _, i := node.Locate(filter.Key); i != -1 {
				values := node.Value.([]*Node)[i]
				if filter.Type == FilterList {
					if values.Type == NodeList {
						for _, v := range values.Value.([]*Node) {
							remapNodes(v, keys[1:], value)
						}
					}
				} else if values.Type == NodeDict {
					remapNodes(values, keys[1:], value)
				}
			}
		}
	}

	// last := len(keys) - 1
	// key := keys[i]
	// if node.Key != "" {
	// 	if key != node.Key {
	// 		node = nil
	// 		break
	// 	}
	// 	if i == last {

	// 	}
	// }
}
