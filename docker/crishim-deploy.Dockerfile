FROM golang:1.11 as build
COPY . /go/src/github.com/Microsoft/KubeGPU/
RUN export CGO_LDFLAGS_ALLOW='-Wl,--unresolved-symbols=ignore-in-object-files' && \
    go build -o /go/bin/crishim github.com/Microsoft/KubeGPU/crishim/cmd && \
    go build --buildmode=plugin -o /go/bin/nvidiagpuplugin.so github.com/Microsoft/KubeGPU/plugins/nvidiagpuplugin/plugin

FROM alpine
COPY --from=build /go/bin/crishim /
COPY --from=build /go/bin/nvidiagpuplugin.so /
COPY docker/crishim-deploy.sh /
CMD ["sh", "/crishim-deploy.sh"]
