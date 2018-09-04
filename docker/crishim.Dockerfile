FROM golang:1.11 as build
COPY . /go/src/github.com/Microsoft/KubeGPU/
RUN export CGO_LDFLAGS_ALLOW='-Wl,--unresolved-symbols=ignore-in-object-files' && \
    go build -o /go/bin/crishim github.com/Microsoft/KubeGPU/crishim/cmd && \
    go build --buildmode=plugin -o /go/bin/nvidiagpuplugin.so github.com/Microsoft/KubeGPU/plugins/nvidiagpuplugin/plugin

FROM debian:stretch-slim
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=utility
COPY --from=build /go/bin/crishim /usr/local/bin/
COPY --from=build /go/bin/nvidiagpuplugin.so /usr/local/KubeExt/devices/
CMD ["crishim", "-v=3"]
