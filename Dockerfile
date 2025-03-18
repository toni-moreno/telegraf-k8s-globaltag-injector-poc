ARG GO_VERSION=1.24
 
# STAGE 1: building the executable
FROM golang:${GO_VERSION}-alpine AS build
RUN apk add --no-cache git
RUN apk --no-cache add ca-certificates
 
# add a user here because addgroup and adduser are not available in scratch
#RUN addgroup -S myapp \
#    && adduser -S -u 10000 -g myapp myapp
 
WORKDIR /src
COPY ./go.mod ./go.sum ./
COPY main.go ./main.go
RUN go mod download
 
#COPY ./ ./
 
# Run tests
#RUN CGO_ENABLED=0 go test -timeout 30s -v github.com/gbaeke/go-template/pkg/api
 
# Build the executable
RUN CGO_ENABLED=0 go build  -a -v \
    -installsuffix 'static' \
    -o /k8s-node-label-extrator .
 
# STAGE 2: build the container to run
FROM scratch AS final
LABEL maintainer="toni-moreno"
COPY --from=build /k8s-node-label-extrator /k8s-node-label-extrator
 
# copy ca certs
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
 
# copy users from builder (use from=0 for illustration purposes)
#COPY --from=0 /etc/passwd /etc/passwd
 
#USER myapp
 
CMD ["/k8s-node-label-extrator"]