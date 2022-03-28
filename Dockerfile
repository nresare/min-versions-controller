FROM golang:1.17-bullseye as build

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /go/bin/min-versions-controller ./cmd/min-versions-controller

RUN apt update && apt -y install upx-ucl
RUN upx /go/bin/min-versions-controller

FROM gcr.io/distroless/static-debian11
USER nobody
COPY --from=build /go/bin/min-versions-controller /
CMD ["/min-versions-controller"]
