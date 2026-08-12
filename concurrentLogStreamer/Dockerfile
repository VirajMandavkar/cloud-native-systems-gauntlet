#=====================================================
# STAGE 1 : Build the executable
#=====================================================
FROM golang:1.22-bookworm AS builder

# Sets the working directroy inside the container
WORKDIR /app

# Copying the source code into the container
COPY main.go .

# Commpiling the Go application into a static binary named "log-aggrigator"
RUN CGO_ENABLED=0 GOOS=linux go build -o log-aggregator main.go

# ==========================================
# STAGE 2: Build the final lightweight image
# ==========================================
FROM debian:bookworm-slim

WORKDIR /app

# Pulling the compiled binary from the 'builder' stage above
COPY --from=builder /app/log-aggregator .

# Telling the Docker this container uses port 8080
EXPOSE 8080

# command to run when the container starts
CMD ["./log-aggregator"]