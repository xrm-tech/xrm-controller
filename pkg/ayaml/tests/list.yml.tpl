dr_import_storages:
- dr_domain_type: fcp
  dr_wipe_after_delete: False
  dr_backup: True
  # Fill in the empty properties related to the secondary site
- dr_domain_type: nfs
- dr_domain_type: iscsi
  dr_wipe_after_delete: True
  dr_backup: False
