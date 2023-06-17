# xrm-controller 

xrm-controller  is a wrapper backend against virtualization site failover procedures.

Focued to oVirt at now.


## Build

`make build`


## Build docker image

`docker build -t xrmtech/xrm-comntroller:latest .`

## Configure (for docker)

Docker container can be configured via environment variables

`docker run -p 8080:8080 -e "XRM_CONTROLLER_USERS=admin:admin" -d xrm-tech/xrm-controller:latest`

## API

[OVirt](./app/xrm-controller/ovirt.md)
