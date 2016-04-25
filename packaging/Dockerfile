FROM alpine:3.2
MAINTAINER Rancher Labs, Inc.
RUN apk add --update ca-certificates 

ENV EXT_LB_RELEASE v0.1.0
ADD https://github.com/rancher/external-lb/releases/download/${EXT_LB_RELEASE}/external-lb.tar.gz /external-lb.tar.gz
RUN tar -zxvf /external-lb.tar.gz -C /usr/bin


ENTRYPOINT ["/usr/bin/external-lb"]
