FROM golang:1.16.0 as builder
ARG SHORT_PACKAGE=rmproxy
ARG PACKAGE=github.com/kwkoo/broadlinkrm
LABEL builder=true
COPY src /go/src/
RUN set -x && \
	cd /go/src/cmd/${SHORT_PACKAGE} && \
	CGO_ENABLED=0 GOOS=linux go build \
		-a \
		-installsuffix cgo \
		-o /go/bin/${SHORT_PACKAGE} \
		.

FROM scratch
LABEL maintainer="kin.wai.koo@gmail.com"
LABEL builder=false
COPY --from=builder /go/bin/${SHORT_PACKAGE} /
COPY json/ /json

# we need to copy the certificates over because we're connecting over SSL
COPY --from=builder /etc/ssl /etc/ssl

# copy timezone info
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

EXPOSE 8080
ENTRYPOINT ["/rmproxy"]
