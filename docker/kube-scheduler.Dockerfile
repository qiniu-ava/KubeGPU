FROM golang:alpine3.7 AS builder
COPY . /go/src/github.com/Microsoft/KubeGPU/
RUN apk add --no-cache build-base
RUN go build -o /kube-scheduler /go/src/github.com/Microsoft/KubeGPU/kube-scheduler/cmd/scheduler.go 
RUN go build --buildmode=plugin -o /gpuschedulerplugin.so /go/src/github.com/Microsoft/KubeGPU/plugins/gpuschedulerplugin/plugin/gpuscheduler.go

FROM alpine:3.7
COPY --from=builder /kube-scheduler /
COPY --from=builder /gpuschedulerplugin.so /schedulerplugins/
ENTRYPOINT [ "/kube-scheduler" ]
