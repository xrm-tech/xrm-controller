FROM  golang:1.20.5 AS builder

COPY ./ /go/src/github.com/xrm-tech/xrm-controller

WORKDIR /go/src/github.com/xrm-tech/xrm-controller
RUN CGO_ENABLED=0 make build


FROM almalinux:8.8

RUN dnf install -y http://resources.ovirt.org/pub/yum-repo/ovirt-release44.rpm
RUN dnf install -y python3-ovirt-engine-sdk4 python3-cryptography
RUN pip3 install ansible-core==2.11.12
RUN ansible-galaxy collection install ovirt.ovirt

RUN mkdir -p /var/lib/xrm-controller/ovirt/template
WORKDIR /root

COPY --from=builder /go/src/github.com/xrm-tech/xrm-controller/xrm-controller ./
COPY ./ovirt/template /var/lib/xrm-controller/ovirt/template

CMD ["./xrm-controller"]
