package ayaml

import (
	"regexp"
	"strconv"
	"strings"
)

type FilterType uint8

const (
	FilterNil FilterType = iota
	FilterKey
	FilterList
	FilterListItem
)

type Filter struct {
	Type FilterType
	Key  string
	Item int // ListItem
}

var (
	reListItem = regexp.MustCompile(`^(.*) *\[ *([0-9]+) *\]$`)
)

func newFilter(key string) Filter {
	key = strings.Trim(key, " ")
	if strings.HasSuffix(key, "[]") {
		key = strings.TrimRight(key[:len(key)-2], " ")
		return Filter{Type: FilterList, Key: key}
	} else {
		if matched := reListItem.FindAllStringSubmatch(key, -1); len(matched) == 1 && len(matched[0]) == 3 {
			n, _ := strconv.Atoi(matched[0][2])
			return Filter{
				Type: FilterListItem,
				Key:  matched[0][1],
				Item: n,
			}
		}
	}

	return Filter{Type: FilterKey, Key: key}
}

// Rewrite allow to rewrite/delete nodes
// use ~ for delete
// [] for iterate over al list items
// [N] for modify one item
// Examples
// "dr_role_mappings[].secondary_name": "~" for delete
// "dr_role_mappings[1]":               "~" for delete
// "dr_role_mappings[2]":               "update"
// "dr_lun_mappings":                   "~"  for delete
// "dr_role_mappings[0].primary_name":  "PRIMARY"
func Rewrite(nodes *Node, remap []string) {
	for _, r := range remap {
		k, v, ok := strings.Cut(r, "=")
		if ok {
			keys := strings.Split(k, ".")
			if len(keys) == 0 || keys[0] == "" {
				continue
			}

			rewriteNodes(nodes, keys, v)
		}
	}
}

func rewriteNodes(node *Node, keys []string, value string) {
	if len(keys) == 0 {
		return
	}

	if values, ok := node.Value.([]*Node); ok {
		if len(keys) == 1 {
			filter := newFilter(keys[0])
			switch filter.Type {
			case FilterList:
				if _, i := node.Locate(filter.Key); i != -1 {
					val := values[i]
					vals := val.Value.([]*Node)
					for j := 0; j < len(vals); j++ {
						if value == "~" {
							val.DeleteItem(j)
						} else {
							val.SetItem(j, value)
						}
					}
				}
			case FilterListItem:
				if _, i := node.Locate(filter.Key); i != -1 {
					v := values[i]
					if value == "~" {
						v.DeleteItem(filter.Item)
					} else {
						v.SetItem(filter.Item, value)
					}
				}
			default:
				if value == "~" {
					node.Delete(keys[0])
				} else {
					node.Set(keys[0], value)
				}
			}
		} else {
			filter := newFilter(keys[0])
			if _, i := node.Locate(filter.Key); i != -1 {
				valueN := values[i]
				switch filter.Type {
				case FilterList:
					if valueN.Type == NodeList {
						for _, v := range valueN.Value.([]*Node) {
							rewriteNodes(v, keys[1:], value)
						}
					}
				case FilterListItem:
					if valueN.Type == NodeList {
						subNode := valueN.Value.([]*Node)
						if filter.Item < len(subNode) {
							rewriteNodes(subNode[filter.Item], keys[1:], value)
						}
					}
				default:
					if valueN.Type == NodeDict {
						rewriteNodes(valueN, keys[1:], value)
					}
				}
			}
		}
	}
}
