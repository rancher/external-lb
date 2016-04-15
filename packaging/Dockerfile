FROM alpine:3.2
MAINTAINER Rancher Labs, Inc.
RUN apk add --update ca-certificates 

ENV EXT_DNS_RELEASE v0.1.0
ADD external-lb /usr/bin/external-lb


ENTRYPOINT ["/usr/bin/external-lb"]
