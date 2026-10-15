FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY rules ./rules
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/client ./cmd/client

FROM scratch AS server
COPY --from=build /out/server /server
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]

FROM scratch AS client
COPY --from=build /out/client /client
COPY examples /examples
USER 65532:65532
ENTRYPOINT ["/client"]
CMD ["-input", "/examples/input.json", "-server", "http://server:8080"]
