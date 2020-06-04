FROM golang:1.14-alpine as builder
WORKDIR /build
COPY . .
# Thanks to cgo sqlite we need gcc and co. 😩
RUN apk add build-base
RUN go build .

FROM alpine:latest
COPY --from=builder /build/shoppingcart /shoppingcart
COPY --from=builder /build/testdata/db.sqlite3 /data/db.sqlite3
EXPOSE 5000
VOLUME ["/data"]
ENTRYPOINT ["/shoppingcart"]
CMD ["-dsn", "file:/data/db.sqlite3"]
