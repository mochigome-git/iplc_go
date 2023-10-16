# Stage 1: Build the Go program
FROM golang:1.20-alpine AS build
WORKDIR /opt/inkjet-PLCcapture-go

# Copy the project files and build the program
COPY . .
RUN apk --no-cache add gcc musl-dev
RUN cd lib && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mainlib main.go

# Stage 2: Copy the built Go program into a minimal container
FROM alpine:3.14
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /opt/inkjet-PLCcapture-go/lib/mainlib /app/
COPY lib/.env.local /app/.env.local

RUN chmod +x /app/mainlib

CMD ["/app/mainlib"]

# Build Image with command
# docker build -t ij-mt-msp:${version} .
