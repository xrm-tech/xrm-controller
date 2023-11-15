package ayaml

import (
	"strings"
	"testing"
)

func TestRewrite(t *testing.T) {
	tests := []struct {
		name    string
		in      *Node
		rewrite map[string]string
		want    *Node
	}{
		{
			name: "rewrite",
			in: &Node{
				Type: NodeDict,
				Value: []*Node{
					{
						Type: NodeList, Key: "dr_domain_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "internal-authz", Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
									{Type: NodeString, Key: "secondary_name", Value: "internal-authz", Commented: true, Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"},
							},
						},
					},
					{
						Type: NodeList, Key: "dr_role_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "primary_name", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
									{Type: NodeDict, Key: "secondary_name", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "delete", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "set", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
						},
					},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
					{Type: NodeDict, Key: "dr_lun_mappings"},
					{
						Type: NodeList, Key: "list_rewrite", Value: []*Node{
							{Type: NodeString, Value: "item1", Parent: &Node{Type: NodeList, Key: "list_rewrite"}},
							{Type: NodeString, Value: "item2", Parent: &Node{Type: NodeList, Key: "list_rewrite"}},
						},
					},
					{
						Type: NodeList, Key: "list_delete", Value: []*Node{
							{Type: NodeString, Value: "item1", Parent: &Node{Type: NodeList, Key: "list_delete"}},
							{Type: NodeString, Value: "item2", Parent: &Node{Type: NodeList, Key: "list_delete"}},
						},
					},
				},
			},
			rewrite: map[string]string{
				"dr_role_mappings[].secondary_name": "~", // delete
				"dr_role_mappings[1]":               "~", // delete
				"dr_role_mappings[2]":               "update",
				"dr_lun_mappings":                   "~", // delete
				"dr_role_mappings[0].primary_name":  "PRIMARY",
				"list_rewrite[]":                    "update_item",
				"list_delete[]":                     "~", // delete
			},
			want: &Node{
				Type: NodeDict,
				Value: []*Node{
					{
						Type: NodeList, Key: "dr_domain_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "internal-authz", Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
									{Type: NodeString, Key: "secondary_name", Value: "internal-authz", Commented: true, Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"},
							},
						},
					},
					{
						Type: NodeList, Key: "dr_role_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "PRIMARY", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
									{Type: NodeDict, Key: "secondary_name", Deleted: true, Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "delete", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Deleted: true,
								Parent:  &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{Type: NodeString, Value: "update", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
						},
					},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
					{Type: NodeDict, Key: "dr_lun_mappings", Deleted: true},
					{
						Type: NodeList, Key: "list_rewrite", Value: []*Node{
							{Type: NodeString, Value: "update_item", Parent: &Node{Type: NodeList, Key: "list_rewrite"}},
							{Type: NodeString, Value: "update_item", Parent: &Node{Type: NodeList, Key: "list_rewrite"}},
						},
					},
					{
						Type: NodeList, Key: "list_delete", Value: []*Node{
							{Type: NodeString, Value: "item1", Deleted: true, Parent: &Node{Type: NodeList, Key: "list_delete"}},
							{Type: NodeString, Value: "item2", Deleted: true, Parent: &Node{Type: NodeList, Key: "list_delete"}},
						},
					},
				},
			},
		},
		{
			name: "complex",
			in: &Node{
				Type: NodeDict,
				Value: []*Node{
					{Type: NodeLine, Key: "# Please fill in the following properties for the secondary site: "},
					{Type: NodeString, Key: "dr_sites_secondary_url", Value: "https://saengine.localdomain/ovirt-engine/api", Commented: true},
					{Type: NodeString, Key: "dr_sites_secondary_username", Value: "admin@internal", Commented: true},
					{Type: NodeLine, Key: ""},
					{
						Type: NodeList, Key: "dr_cluster_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "Default", Parent: &Node{Type: NodeList, Key: "dr_cluster_mappings"}},
									{Type: NodeString, Key: "secondary_name", Value: "Default", Commented: true, Parent: &Node{Type: NodeList, Key: "dr_cluster_mappings"}},
								},
							},
						},
						Parent: &Node{Type: NodeList, Key: "dr_cluster_mappings"},
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
									{Type: NodeString, Key: "primary_name", Value: "internal-authz", Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
									{Type: NodeString, Key: "secondary_name", Value: "internal-authz", Commented: true, Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
								},
							},
						},
						Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"},
					},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: ""},
					{Type: NodeLine, Key: "# Mapping for role"},
					{Type: NodeLine, Key: "# Fill in any roles which should be mapped between sites."},
					{
						Type: NodeList, Key: "dr_role_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "primary_name", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
									{Type: NodeDict, Key: "secondary_name", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "delete", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "set", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
						},
					},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
					{Type: NodeDict, Key: "dr_lun_mappings"},
				},
			},
			rewrite: map[string]string{
				"dr_sites_secondary_username":       "test@internal",
				"dr_role_mappings[].secondary_name": "~", // delete
				"dr_role_mappings[1]":               "~", // delete
				"dr_role_mappings[2]":               "update",
				"dr_lun_mappings":                   "~", // delete
			},
			want: &Node{
				Type: NodeDict,
				Value: []*Node{
					{Type: NodeLine, Key: "# Please fill in the following properties for the secondary site: "},
					{Type: NodeString, Key: "dr_sites_secondary_url", Value: "https://saengine.localdomain/ovirt-engine/api", Commented: true},
					{Type: NodeString, Key: "dr_sites_secondary_username", Value: "test@internal", Commented: true},
					{Type: NodeLine, Key: ""},
					{
						Type: NodeList, Key: "dr_cluster_mappings", Value: []*Node{
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeString, Key: "primary_name", Value: "Default"},
									{Type: NodeString, Key: "secondary_name", Value: "Default", Commented: true},
								},
								Parent: &Node{Type: NodeList, Key: "dr_cluster_mappings"},
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
									{Type: NodeString, Key: "primary_name", Value: "internal-authz", Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
									{Type: NodeString, Key: "secondary_name", Value: "internal-authz", Commented: true, Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"},
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
									{Type: NodeDict, Key: "primary_name", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
									{Type: NodeDict, Key: "secondary_name", Deleted: true, Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeDict, Value: []*Node{
									{Type: NodeDict, Key: "delete", Parent: &Node{Type: NodeList, Key: "dr_role_mappings"}},
								},
								Deleted: true,
								Parent:  &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
							{
								Type: NodeString, Value: "update",
								Parent: &Node{Type: NodeList, Key: "dr_role_mappings"},
							},
						},
					},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
					{Type: NodeDict, Key: "dr_lun_mappings", Deleted: true},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Rewrite(tt.in, tt.rewrite)

			var diff []string
			diffNodes(tt.in, tt.want, "", -1, &diff)
			if len(diff) > 0 {
				t.Errorf("Rewrite() =\n%s", strings.Join(diff, "\n"))
			}
		})
	}
}
