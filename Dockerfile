FROM golang:1.10.3 as builder
ARG PACKAGE=golang_broadlink_rm
LABEL builder=true
COPY src /go/src/
RUN set -x && \
	cd /go/src/${PACKAGE}/cmd/${PACKAGE} && \
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/${PACKAGE} .

FROM scratch
LABEL maintainer="glug71@gmail.com"
LABEL builder=false
COPY --from=builder /go/bin/${PACKAGE} /

# we need to copy the certificates over because we're connecting over SSL
COPY --from=builder /etc/ssl /etc/ssl

EXPOSE 8080

ENTRYPOINT ["/golang-broadlink-rm"]
