FROM golang:1.21-alpine AS build
WORKDIR /src
COPY src/cli/go.mod src/cli/go.sum ./
RUN go mod download
COPY src/cli /src
RUN go build -o /usr/local/bin/isp-checker .

FROM alpine:3.18
COPY --from=build /usr/local/bin/isp-checker /usr/local/bin/isp-checker
ENTRYPOINT ["/usr/local/bin/isp-checker"]
