package ayaml

import (
	"strings"
	"testing"
)

func TestRemap(t *testing.T) {
	tests := []struct {
		name  string
		in    *Node
		remap map[string]string
		want  *Node
	}{
		{
			name: "",
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
									{Type: NodeString, Key: "primary_name", Value: "Default"},
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
								Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"},
							},
						},
					},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
					{Type: NodeDict, Key: "dr_lun_mappings"},
				},
			},
			remap: map[string]string{
				"dr_sites_secondary_username":       "test@internal",
				"dr_role_mappings[].secondary_name": "~", // delete
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
								},
								Parent: &Node{Type: NodeList, Key: "dr_domain_mappings"},
							},
						},
					},
					{Type: NodeLine, Key: "# Mapping for external LUN disks"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Remap(tt.in, tt.remap)

			var diff []string
			diffNodes(tt.in, tt.want, "", -1, &diff)
			if len(diff) > 0 {
				t.Errorf("Remap() =\n%s", strings.Join(diff, "\n"))
			}
		})
	}
}
