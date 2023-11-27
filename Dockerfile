FROM golang:1.21-alpine AS golang

RUN apk add --no-cache ca-certificates git

ADD go.mod go.sum /usr/src/
WORKDIR /usr/src
RUN go mod download

ADD main.go /usr/src/
ADD pkg/ /usr/src/pkg/
RUN CGO_ENABLED=0 go build -a -ldflags '-w' -o ./terraform-bucket-registry ./main.go

FROM scratch

COPY --from=golang /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=golang /usr/src/terraform-bucket-registry /

ENTRYPOINT ["/terraform-bucket-registry"]
