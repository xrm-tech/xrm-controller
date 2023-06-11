FROM  golang:1.20.5 AS builder

COPY ./ /go/src/github.com/xrm-tech/xrm-controller

WORKDIR /go/src/github.com/xrm-tech/xrm-controller
RUN CGO_ENABLED=0 make build


FROM  alpine:3.18.0

RUN apk --no-cache add ca-certificates
RUN apk add python3 py3-pip
RUN pip3 install ansible-core==2.12.3
RUN ansible-galaxy install ovirt.engine-setup

RUN mkdir -p /var/lib/xrm-controller/ovirt/template
WORKDIR /root

COPY --from=builder /go/src/github.com/xrm-tech/xrm-controller/xrm-controller ./
COPY ./ovirt/template /var/lib/xrm-controller/ovirt/template

CMD ["./xrm-controller"]
