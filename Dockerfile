FROM golang:1.24.2 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' -o /out/node-pinger ./cmd/node-pinger

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/node-pinger /node-pinger

EXPOSE 9095

ENTRYPOINT ["/node-pinger"]
