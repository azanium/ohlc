# Use the official Alpine-based Golang image
FROM golang:1.23.3-alpine

# Add necessary build dependencies
RUN apk add --no-cache gcc musl-dev

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o ohlc ./cmd/ohlc

# Set the environment variables for PostgreSQL connection
ENV POSTGRES_USER=your_user
ENV POSTGRES_PASSWORD=your_password
ENV POSTGRES_DB=your_db
ENV POSTGRES_HOST=your_host
ENV POSTGRES_PORT=5432

# Specify the entry point for the container
CMD ["./ohlc"]