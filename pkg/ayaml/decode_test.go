package ayaml

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func diffNodes(got, want *Node, indent string, arrayN int, diff *[]string) {
	if got != want {
		if got == nil {
			*diff = append(*diff, fmt.Sprintf("%s-   %+v", indent, want))
		} else if want == nil {
			*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
			*diff = append(*diff, fmt.Sprintf("%s+   %+v", indent, got))
		} else {
			var different bool
			if got.Key != want.Key {
				if !different {
					different = true
					*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
				}
				*diff = append(*diff, fmt.Sprintf("%s    Key %q != %q", indent, got.Key, want.Key))
			}
			if got.Commented != want.Commented {
				if !different {
					different = true
					*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
				}
				*diff = append(*diff, fmt.Sprintf("%s    Commented %v != %v", indent, got.Commented, want.Commented))
			}

			if got.Type != want.Type {
				if !different {
					// different = true
					*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
				}
				*diff = append(*diff, fmt.Sprintf("%s    Type %q != %q", indent, got.Type.String(), want.Type.String()))
			} else {
				switch gotV := got.Value.(type) {
				case nil:
					if want.Value != nil {
						*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
						if wantV, ok := want.Value.([]*Node); !ok {
							for i := 0; i < len(wantV); i++ {
								*diff = append(*diff, fmt.Sprintf("%s-     %+v", indent, wantV))
							}
						} else {
							for i := 0; i < len(wantV); i++ {
								*diff = append(*diff, fmt.Sprintf("%s-    [%2d] %+v", indent, i, wantV[i]))
							}
						}
					}
				case string:
					if wantV, ok := want.Value.(string); !ok {
						if !different {
							// different = true
							*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
						}
						*diff = append(*diff, fmt.Sprintf("%s    Value %q != %+v", indent, gotV, wantV))
					} else if gotV != wantV {
						if !different {
							// different = true
							*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
						}
						*diff = append(*diff, fmt.Sprintf("%s    Value %q != %q", indent, gotV, wantV))
					}
				case []*Node:
					if wantV, ok := want.Value.([]*Node); !ok {
						if !different {
							// different = true
							*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
						}
						// *diff = append(*diff, fmt.Sprintf("%s    Value %+v != %+v", indent, gotV, wantV))
						for i := 0; i < len(gotV); i++ {
							*diff = append(*diff, fmt.Sprintf("%s+    [%2d] %+v", indent, i, gotV[i]))
						}
					} else {
						n := max(len(gotV), len(wantV))
						for i := 0; i < n; i++ {
							if i >= len(gotV) {
								if !different {
									different = true
									*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
								}
								*diff = append(*diff, fmt.Sprintf("%s-    [%2d] %+v", indent, i, wantV[i]))
							} else if i >= len(wantV) {
								if !different {
									different = true
									*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
								}
								*diff = append(*diff, fmt.Sprintf("%s+    [%2d] %+v", indent, i, gotV[i]))
							} else {
								diffNodes(gotV[i], wantV[i], indent+" ", i, diff)
							}
						}
					}
				case *Node:
					if wantV, ok := want.Value.(*Node); !ok {
						if !different {
							// different = true
							*diff = append(*diff, fmt.Sprintf("---%s (line %d)", got.ParentKey(), got.Line))
						}
						*diff = append(*diff, fmt.Sprintf("%s+      %+v", indent, gotV))
					} else {
						diffNodes(gotV, wantV, indent+" ", -1, diff)
					}
				default:
					panic(fmt.Errorf("unhandled: %+v", got))
				}
			}
		}
	}
}

func Test_nextUncommented(t *testing.T) {
	tests := []struct {
		name        string
		items       []*item
		start       int
		indent      int
		wantPos     int
		wantComment int
	}{
		{
			name: "merge0",
			items: []*item{
				{Key: "mappings", Line: 1},
				{Key: "first", Value: "test", ListStart: true, Indent: 2, Line: 2},
				{Key: "second", Value: "test", Indent: 2, Line: 3},
				{Key: "  # last", Comment: true, Line: 4},
				{Key: "# first,", Comment: true, Line: 5},
				{Key: "# second", Comment: true, Line: 6},
				{Comment: true, Line: 7},
				{Key: "# third", Comment: true, Line: 8},
				{Key: "third", Value: "test", Line: 9},
			},
			start: 3, indent: 0, wantPos: 8, wantComment: 3,
		},
		{
			name: "merge_blank",
			items: []*item{
				{Key: "mappings", Line: 1},
				{Key: "first", Value: "test", ListStart: true, Indent: 2, Line: 2},
				{Key: "second", Value: "test", Indent: 2, Line: 3},
				{Key: "  # last", Comment: true, Line: 4},
				{Comment: true, Line: 7},
				{Key: "# third", Comment: true, Line: 8},
				{Key: "third", Value: "test", Line: 9},
			},
			start: 3, indent: 0, wantPos: 6, wantComment: 3,
		},
		{
			name: "merge1",
			items: []*item{
				{Key: "mappings", Line: 1},
				{Key: "first", Value: "test", ListStart: true, Indent: 2, Line: 2},
				{Key: "second", Value: "test", Indent: 2, Line: 3},
				{Key: "  # last", Comment: true, Line: 4},
				{Key: "  # third", Comment: true, Line: 8},
				{Key: "third", Value: "test", Line: 9},
			},
			start: 3, indent: 0, wantPos: 5, wantComment: 3,
		},
		{
			name: "merge_no",
			items: []*item{
				{Key: "mappings", Line: 1},
				{Key: "first", Value: "test", ListStart: true, Indent: 2, Line: 2},
				{Key: "second", Value: "test", Indent: 2, Line: 3},
				{Key: "third", Value: "test", Line: 9},
			},
			start: 3, indent: 0, wantPos: 3, wantComment: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, comment := nextUncommented(tt.items, tt.start)
			if pos != tt.wantPos {
				t.Errorf("nextUncommented().pos = %v, want %v", pos, tt.wantPos)
			}
			if comment != tt.wantComment {
				t.Errorf("nextUncommented().comment = %v, want %v", pos, tt.wantComment)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		inPath   string
		wantNode *Node
		wantErr  bool
	}{
		{
			inPath: "merge.yml.tpl",
			wantNode: &Node{
				Type: NodeDict,
				Value: []*Node{
					{
						Type: NodeList, Key: "mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "first", Value: "test"},
									{Type: NodeString, Key: "second", Value: "test"},
								},
							},
						},
					},
					{Type: NodeLine, Key: "  # last"},
					{Type: NodeLine, Key: "# first"},
					{Type: NodeLine, Key: "# second"},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# third"},
					{Type: NodeString, Key: "third", Value: "test"},
				},
			},
		},
		{
			inPath: "dict.yml.tpl",
			wantNode: &Node{
				Type: NodeDict,
				Value: []*Node{
					{
						Type: NodeList, Key: "dr_network_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_network_name", Value: "ovirtmgmt"},
									{Type: NodeLine, Key: "# Data Center name is relevant when multiple vnic profiles are maintained."},
									{Type: NodeLine, Key: "# please uncomment it in case you have more than one DC."},
									{Type: NodeLine, Key: "# primary_network_dc: Default"},
									{Type: NodeString, Key: "primary_profile_name", Value: "ovirtmgmt"},
									{Type: NodeString, Key: "primary_profile_id", Value: "657e2905-1b6a-4647-a98d-0e1c261b3024"},
									{Type: NodeLine, Key: "  # Fill in the correlated vnic profile properties in the secondary site for profile 'ovirtmgmt'"},
									{Type: NodeString, Key: "secondary_network_name", Value: "ovirtmgmt", Commented: true},
									{Type: NodeLine, Key: "# Data Center name is relevant when multiple vnic profiles are maintained."},
									{Type: NodeLine, Key: "# please uncomment it in case you have more than one DC."},
									{Type: NodeLine, Key: "# secondary_network_dc: Default"},
									{Type: NodeString, Key: "secondary_profile_name", Value: "ovirtmgmt", Commented: true},
									{Type: NodeString, Key: "secondary_profile_id", Value: "657e2905-1b6a-4647-a98d-0e1c261b3024", Commented: true},
								},
							},
						},
					},
				},
			},
		},
		{
			inPath: "dict_empty.yml.tpl",
			wantNode: &Node{
				Type: NodeDict,
				Value: []*Node{
					{Type: NodeLine, Key: "# Mapping for role"},
					{Type: NodeLine, Key: "# Fill in any roles which should be mapped between sites."},
					{
						Type: NodeList, Key: "dr_role_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "primary_name"},
									{Type: NodeDict, Key: "secondary_name"},
								},
							},
						},
					},
				},
			},
		},
		{
			inPath: "list.yml.tpl",
			wantNode: &Node{
				Type: NodeDict,
				Value: []*Node{
					{
						Type: NodeList, Key: "dr_import_storages", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "dr_domain_type", Value: "fcp"},
									{Type: NodeString, Key: "dr_wipe_after_delete", Value: "False"},
									{Type: NodeString, Key: "dr_backup", Value: "True"},
									{Type: NodeLine, Key: "  # Fill in the empty properties related to the secondary site"},
								},
							},
							{Type: NodeString, Key: "dr_domain_type", Value: "nfs"},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "dr_domain_type", Value: "iscsi"},
									{Type: NodeString, Key: "dr_wipe_after_delete", Value: "True"},
									{Type: NodeString, Key: "dr_backup", Value: "False"},
								},
							},
						},
					},
				},
			},
		},
		{
			inPath: "disaster_recovery_vars.yml.tpl",
			wantNode: &Node{
				Type: NodeDict,
				Value: []*Node{
					{Type: NodeString, Key: "dr_sites_primary_url", Value: "https://saengine.localdomain/ovirt-engine/api"},
					{Type: NodeString, Key: "dr_sites_primary_username", Value: "admin@internal"},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Please fill in the following properties for the secondary site: "},
					{Type: NodeString, Key: "dr_sites_secondary_url", Value: "https://saengine.localdomain/ovirt-engine/api", Commented: true},
					{Type: NodeString, Key: "dr_sites_secondary_username", Value: "admin@internal", Commented: true},
					{Type: NodeLine, Key: ""},
					{
						Type: NodeList, Key: "dr_import_storages", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "dr_domain_type", Value: "fcp"},
									{Type: NodeString, Key: "dr_wipe_after_delete", Value: "False"},
									{Type: NodeString, Key: "dr_backup", Value: "False"},
									{Type: NodeLine, Key: "  # Fill in the empty properties related to the secondary site"},
									{Type: NodeString, Key: "dr_secondary_dc_name", Value: "Default", Commented: true},
								},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "dr_domain_type", Value: "nfs"},
									{Type: NodeString, Key: "dr_wipe_after_delete", Value: "False"},
									{Type: NodeString, Key: "dr_backup", Value: "False"},
									{Type: NodeLine, Key: "  # Fill in the empty properties related to the secondary site"},
								},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "dr_domain_type", Value: "nfs"},
									{Type: NodeLine, Key: "  # Fill in the empty properties related to the secondary site"},
									{Type: NodeString, Key: "dr_secondary_name", Value: "nfs_dom_2", Commented: true},
									{Type: NodeString, Key: "dr_secondary_master_domain", Value: "True", Commented: true},
									{Type: NodeString, Key: "dr_secondary_dc_name", Value: "Default", Commented: true},
									{Type: NodeString, Key: "dr_secondary_path", Value: "/nfs_dom_dr_2/", Commented: true},
									{Type: NodeString, Key: "dr_secondary_address", Value: "10.1.1.2", Commented: true},
								},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "dr_domain_type", Value: "fcp"},
									{Type: NodeLine, Key: "  # Fill in the empty properties related to the secondary site"},
									{Type: NodeString, Key: "dr_wipe_after_delete", Value: "False"},
									{Type: NodeString, Key: "dr_backup", Value: "False"},
								},
							},
						},
					},
					{Type: NodeLine, Key: "  # Fill in the empty properties related to the secondary site"},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for cluster"},
					{
						Type: NodeList, Key: "dr_cluster_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "Default"},
									{Type: NodeLine, Key: "  # Fill the correlated cluster name in the secondary site for cluster 'Default'"},
									{Type: NodeString, Key: "secondary_name", Value: "Default", Commented: true},
								},
							},
						},
					},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for affinity group"},
					{Type: NodeDict, Key: "dr_affinity_group_mappings"},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for affinity label"},
					{Type: NodeDict, Key: "dr_affinity_label_mappings"},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for domain"},
					{
						Type: NodeList, Key: "dr_domain_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "internal-authz"},
									{Type: NodeLine, Key: "  # Fill in the correlated domain in the secondary site for domain 'internal-authz'"},
									{Type: NodeString, Key: "secondary_name", Value: "internal-authz", Commented: true},
								},
							},
						},
					},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for role"},
					{Type: NodeLine, Key: "# Fill in any roles which should be mapped between sites."},
					{
						Type: NodeList, Key: "dr_role_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "primary_name"},
									{Type: NodeDict, Key: "secondary_name"},
								},
							},
						},
					},
					{Type: NodeLine, Key: ""},
					{
						Type: NodeList, Key: "dr_network_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_network_name", Value: "ovirtmgmt"},
									{Type: NodeLine, Key: "# Data Center name is relevant when multiple vnic profiles are maintained."},
									{Type: NodeLine, Key: "# please uncomment it in case you have more than one DC."},
									{Type: NodeLine, Key: "# primary_network_dc: Default"},
									{Type: NodeString, Key: "primary_profile_name", Value: "ovirtmgmt"},
									{Type: NodeString, Key: "primary_profile_id", Value: "657e2905-1b6a-4647-a98d-0e1c261b3024"},
									{Type: NodeLine, Key: "  # Fill in the correlated vnic profile properties in the secondary site for profile 'ovirtmgmt'"},
									{Type: NodeString, Key: "secondary_network_name", Value: "ovirtmgmt", Commented: true},
									{Type: NodeLine, Key: "# Data Center name is relevant when multiple vnic profiles are maintained."},
									{Type: NodeLine, Key: "# please uncomment it in case you have more than one DC."},
									{Type: NodeLine, Key: "# secondary_network_dc: Default"},
									{Type: NodeString, Key: "secondary_profile_name", Value: "ovirtmgmt", Commented: true},
									{Type: NodeString, Key: "secondary_profile_id", Value: "657e2905-1b6a-4647-a98d-0e1c261b3024", Commented: true},
								},
							},
						},
					},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
					{Type: NodeDict, Key: "dr_lun_mappings"},
				},
			},
		},
	}

	_, filename, _, _ := runtime.Caller(0)
	testDir := path.Join(path.Dir(filename), "tests")

	for _, tt := range tests {
		t.Run(tt.inPath, func(t *testing.T) {
			path := path.Join(testDir, tt.inPath)
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}

			node, err := Decode(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var diff []string
			diffNodes(node, tt.wantNode, "", -1, &diff)
			if len(diff) > 0 {
				t.Errorf("Decode() =\n%s", strings.Join(diff, "\n"))
			}
		})
	}
}
