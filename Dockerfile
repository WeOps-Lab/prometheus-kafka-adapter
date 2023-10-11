FROM golang:1.19.1-alpine as build
WORKDIR /src/prometheus-kafka-adapter

COPY go.mod go.sum *.go ./

ADD . /src/prometheus-kafka-adapter

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apk/repositories && \
    apk add gcc musl-dev && \
    go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct

RUN go mod tidy && \
    go mod vendor && \
    go build -ldflags='-w -s -extldflags "-static"' -tags musl,static,netgo -mod=vendor -o /prometheus-kafka-adapter

FROM alpine:3.16

COPY schemas/metric.avsc /schemas/metric.avsc
COPY metrics.yaml /metrics.yaml
COPY --from=build /prometheus-kafka-adapter /

CMD /prometheus-kafka-adapter