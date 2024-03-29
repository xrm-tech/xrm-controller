---
dr_sites_primary_url: https://saengine.localdomain/ovirt-engine/api
dr_sites_primary_username: admin@internal
dr_sites_primary_ca_file: /home/user/go/src/github.com/xrm-tech/xrm-controller/ovirt/test/primary.ca

# Please fill in the following properties for the secondary site: 
dr_sites_secondary_url: # https://saengine.localdomain/ovirt-engine/api
dr_sites_secondary_username: # admin@internal
dr_sites_secondary_ca_file: # /var/lib/xrm-controller/ovirt/test/primary.ca

dr_import_storages:
- dr_domain_type: nfs
  dr_wipe_after_delete: False
  dr_backup: False
  dr_critical_space_action_blocker: 5
  dr_storage_domain_type: data
  dr_warning_low_space: 10
  dr_primary_name: nfs_dom
  dr_primary_master_domain: True
  dr_primary_dc_name: Default
  dr_primary_path: /nfs_dom_dr/
  dr_primary_address: 10.1.1.2
  # Fill in the empty properties related to the secondary site
  dr_secondary_name: # nfs_dom
  dr_secondary_master_domain: # True
  dr_secondary_dc_name: # Default
  dr_secondary_path: # /nfs_dom_dr/
  dr_secondary_address: # 10.1.1.2
- dr_domain_type: nfs
  dr_wipe_after_delete: False
  dr_backup: False
  dr_critical_space_action_blocker: 5
  dr_storage_domain_type: data
  dr_warning_low_space: 10
  dr_primary_name: nfs_dom_2
  dr_primary_master_domain: True
  dr_primary_dc_name: Default
  dr_primary_path: /nfs_dom_dr_2/
  dr_primary_address: 10.1.1.2
  # Fill in the empty properties related to the secondary site
  dr_secondary_name: # nfs_dom_2
  dr_secondary_master_domain: # True
  dr_secondary_dc_name: # Default
  dr_secondary_path: # /nfs_dom_dr_2/
  dr_secondary_address: # 10.1.1.2

# Mapping for cluster
dr_cluster_mappings:
- primary_name: Default
  # Fill the correlated cluster name in the secondary site for cluster 'Default'
  secondary_name: # Default


# Mapping for affinity group
dr_affinity_group_mappings:

# Mapping for affinity label
dr_affinity_label_mappings:

# Mapping for domain
dr_domain_mappings: 
- primary_name: internal-authz
  # Fill in the correlated domain in the secondary site for domain 'internal-authz'
  secondary_name: # internal-authz
  


# Mapping for role
# Fill in any roles which should be mapped between sites.
dr_role_mappings: 
- primary_name: 
  secondary_name: 

dr_network_mappings:
- primary_network_name: ovirtmgmt
# Data Center name is relevant when multiple vnic profiles are maintained.
# please uncomment it in case you have more than one DC.
# primary_network_dc: Default
  primary_profile_name: ovirtmgmt
  primary_profile_id: 657e2905-1b6a-4647-a98d-0e1c261b3024
  # Fill in the correlated vnic profile properties in the secondary site for profile 'ovirtmgmt'
  secondary_network_name: # ovirtmgmt
# Data Center name is relevant when multiple vnic profiles are maintained.
# please uncomment it in case you have more than one DC.
# secondary_network_dc: Default
  secondary_profile_name: # ovirtmgmt
  secondary_profile_id: # 657e2905-1b6a-4647-a98d-0e1c261b3024


# Mapping for external LUN disks
dr_lun_mappings:
