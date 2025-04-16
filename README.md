# OHLC Trading Chart Service

A real-time OHLC (Open, High, Low, Close) trading chart service that streams cryptocurrency price data using gRPC. The service aggregates price data from Binance WebSocket feeds and provides OHLC candlestick data to clients through a streaming API.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Local Development Setup](#local-development-setup)
- [Running the Client](#running-the-client)
- [API Documentation](#api-documentation)
  - [gRPC Service](#grpc-service)
  - [Subscribe Request](#subscribe-request)
  - [OHLC Data](#ohlc-data)
- [Deployment](#deployment)
  - [Setup Digital Ocean Token](#setup-digital-ocean-token)
  - [Docker](#docker)
  - [Kubernetes Infrastructure Setup with Terraform](#kubernetes-infrastructure-setup-with-terraform)
  - [Install Service to Kubernetes Cluster](#install-service-to-kubernetes-cluster)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Contact](#contact)
- [License](#license)

## Features

- Real-time OHLC data streaming via gRPC
- Support for multiple cryptocurrency pairs (BTCUSDT, ETHUSDT, PEPEUSDT)
- Configurable candlestick intervals
- PostgreSQL storage for historical data
- Kubernetes-ready deployment
- Graceful shutdown handling

## Architecture

The service consists of several components:

- **Binance WebSocket Client**: Connects to Binance's WebSocket API to receive real-time trade data
- **Candlestick Aggregator**: Processes trade data into OHLC candlesticks
- **Storage Layer**: Persists OHLC data in PostgreSQL
- **gRPC Streaming Service**: Provides real-time OHLC data to clients

## Project Structure

```markdown
.
├── cmd/                    # Application entry points
│   ├── client/            # Stream client implementation
│   ├── ohlc/              # Main service executable
│   └── sample/            # Sample code for binance WebSocket client
├── conf/                  # Configuration files
│   ├── dev/              # Development environment configs
│   ├── staging/          # Staging environment configs
│   └── production/       # Production environment configs
├── db/                    # Database related files
│   └── schema.sql        # Database schema definitions
├── deployments/           # Deployment configurations
│   ├── helm/             # Helm charts for Kubernetes
│   └── terraform/        # Infrastructure as code
├── internal/              # Private application code
│   ├── binance/          # Binance WebSocket client
│   ├── candlestick/      # OHLC data processing and aggregation
│   ├── proto/            # Internal protobuf implementations
│   ├── service/          # Core service implementation
│   ├── storage/          # Data persistence layer
│   └── streaming/        # gRPC streaming service
└── proto/                # Protocol buffer definitions
```

## Prerequisites

- Go 1.23.3 or later
- PostgreSQL 13 or later
- Docker (for containerized deployment)
- Docker Compose (for local development)
- Kubernetes (for production deployment)
- Helm (for Kubernetes deployment)
- Terraform (for infrastructure as code)
- DigitalOcean account and API token
- doctl (DigitalOcean CLI)
- Protocol Buffers compiler
- kubectl (optional)

## Local Development Setup

1. Clone the repository
2. Install dependencies:

   ```bash
   go mod download
   ```

3. Set up PostgreSQL database:

   ```bash
   psql -U postgres -f db/schema.sql
   ```

4. Configure the service:
   - Copy `conf/dev/conf.yaml` to your working directory
   - Adjust database and server settings as needed

5. Start the service:

   ```bash
   make run_dev
   ```

## Running the Client

The streaming client can be run with:

```bash
# Default connection to localhost:8080
go run cmd/client/stream_client.go

# Custom service address
OHLC_SERVICE_ADDR=localhost:8080 go run cmd/client/stream_client.go
```

## API Documentation

### gRPC Service

The service provides a streaming API defined in `proto/ohlc.proto`:

```protobuf
service OHLCService {
  rpc StreamOHLC(SubscribeRequest) returns (stream OHLC) {}
}
```

#### Subscribe Request

```protobuf
message SubscribeRequest {
  repeated string symbols = 1;
}
```

#### OHLC Data

```protobuf
message OHLC {
  string symbol = 1;
  double open = 2;
  double high = 3;
  double low = 4;
  double close = 5;
  double volume = 6;
  int64 open_time = 7;
  int64 close_time = 8;
}
```

## Deployment

### Setup Digital Ocean Token

```bash
export DO_TOKEN=<your_digital_ocean_token>
```

### Docker

Build and run using Docker:

```bash
docker build -t your-repo/ohlc .
docker push your-repo/ohlc:latest
```

### Kubernetes Infrastructure Setup with Terraform

1. Install Terraform:

   ```bash
   # For macOS using Homebrew
   brew install terraform

   # For Linux
   wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
   echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
   sudo apt-get update && sudo apt-get install terraform
   ```

2. Initialize Terraform:

   ```bash
   cd deployments/terraform
   terraform init
   ```

3. Review and apply the infrastructure:

   ```bash
   terraform plan -var="do_token=$DO_TOKEN" \
      -var="image=azanium/ohlc:latest" \
      -var="postgres_password=demo123"
   terraform apply -var="do_token=$DO_TOKEN" \
      -var="image=azanium/ohlc:latest" \
      -var="postgres_password=demo123"
   ```

### Install Service to Kubernetes Cluster

Deploy to Kubernetes using Helm:
Go to the project root directory!

```bash
helm upgrade --install ohlc ./deployments/helm/ohlc
```

## Configuration

The service can be configured through environment variables or configuration files:

- `OHLC_SERVICE_ADDR`: gRPC service address (default: ":8080")
- `POSTGRES_*`: Database connection settings
- See `conf/dev/conf.yaml` for all available options

## Monitoring

To monitor the deployed services:

```bash
# View service logs
kubectl logs -f deployment/ohlc -n ohlc

# Check PostgreSQL status
kubectl exec -it deployment/postgres -n ohlc -- psql -U ohlc -d ohlc
```

## Contact

For support, bug reports, or contributions:

syuaibi [at] gmail [dot] com

## License

MIT License
