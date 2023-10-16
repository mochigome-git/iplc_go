# Stage 1: Build the Go program
FROM golang:1.20-alpine AS build
WORKDIR /opt/nk2PLCcapture

# Copy the project files and build the program
COPY . .
RUN apk --no-cache add gcc musl-dev
RUN cd 2bit && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main2bit main.2bit.go

# Stage 2: Copy the built Go program into a minimal container
FROM alpine:3.14
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /opt/nk2PLCcapture/2bit/main2bit /app/
COPY 2bit/.env.local /app/.env.local

RUN chmod +x /app/main2bit

CMD ["/app/main2bit"]
