FROM golang:1.10.3 as builder
ARG SHORT_PACKAGE=rmproxy
ARG PACKAGE=github.com/kwkoo/broadlinkrm
LABEL builder=true
COPY src /go/src/
RUN set -x && \
	cd /go/src/${PACKAGE}/cmd/${SHORT_PACKAGE} && \
	CGO_ENABLED=0 GOOS=linux go build \
		-a \
		-installsuffix cgo \
		-o /go/bin/${SHORT_PACKAGE} \
		.

FROM scratch
LABEL maintainer="glug71@gmail.com"
LABEL builder=false
COPY --from=builder /go/bin/${SHORT_PACKAGE} /
COPY json/ /json

# we need to copy the certificates over because we're connecting over SSL
COPY --from=builder /etc/ssl /etc/ssl

EXPOSE 8080

ENTRYPOINT ["/rmproxy"]
