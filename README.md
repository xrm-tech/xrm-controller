# xrm-controller Ovirt API


## Build

`make build`


## Build docker image

`docker build -t xrm-tech/xrm-comntroller:latest .`

## Configure (for docker)

Docker container can be configured via environment variables

`docker run -p 8080:8080 -e "XRM_CONTROLLER_USERS=admin:admin" -d xrm-tech/xrm-comntroller:latest`

## API

[OVirt](./app/xrm-controller/ovirt.md)
