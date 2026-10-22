FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /grange ./cmd/grange

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /grange /grange
VOLUME /data
EXPOSE 8081
ENV GRANGE_DATA=/data
ENTRYPOINT ["/grange", "serve"]
