# xrm-controller OVirt API

Generate config `/ovirt/generate/:name`

  - site_primary_url

  - site_primary_username
 
  - site_primary_password
 
  - site_secondary_url
 
  - site_secondary_username
 
   - site_secondary_password
 
   - storage_domains

Example:

`curl -i -u admin:password -X POST -H "Content-Type: application/json" -d '{"site_primary_url": "https://engine1.localdomain/ovirt-engine/api", "site_primary_username": "admin@ovirt@internalsso", "site_secondary_url": "https://engine2.localdomain/ovirt-engine/api", "site_secondary_username": "admin@ovirt@internalsso", "site_primary_password": "password", "site_secondary_password": "password", "storage_domains": [{"storage_type": "nfs", "primary_name": "nfstst", "primary_path": "/nfs_tst/", "primary_addr": "192.168.122.210", "secondary_path": "/nfs_tst_replica/", "secondary_addr": "192.168.122.210"}]}' http://127.0.0.1:8080/ovirt/^Cnerate/test`


Delete config `/ovirt/delete/:name`

Failover (for generated config) `/ovirt/failover/:name`
