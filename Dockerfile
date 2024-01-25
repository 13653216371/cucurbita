FROM docker.io/golang AS builder
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0
RUN go build

FROM scratch
VOLUME /var/lib/cucurbita
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/cucurbita /usr/bin/cucurbita
ENTRYPOINT ["/usr/bin/cucurbita"]
