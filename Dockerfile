FROM docker.io/openshift/origin-release:golang-1.15 AS builder
WORKDIR /go/src/github.com/zhiweiyin318/metric-test
COPY . .
ENV GO_PACKAGE github.com/zhiweiyin318/metric-test
RUN make build --warn-undefined-variables

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /go/src/github.com/zhiweiyin318/metric-test/metric-test /
RUN microdnf update && microdnf clean all
