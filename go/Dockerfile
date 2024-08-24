FROM golang:1.21 as builder
WORKDIR /wokrdir

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM alpine:3.20
COPY --from=builder  /wokrdir/main ./main
ENTRYPOINT [ "./main" ]
