# OHLC Trading Chart Service

## Prerequisites

- Go 1.21 or later
- Docker
- Terraform
- DigitalOcean account and API token
- Protocol Buffers compiler
- SQLite3
- kubectl
- doctl (DigitalOcean CLI)

## Local Development Setup

### Getting Started

1. Clone the repository and navigate to the project directory

2. Start the development environment (PostgreSQL and OHLC service):

   ```bash
   make run_dev
   ```

   This command will:
   - Build and start PostgreSQL container
   - Initialize the database with required schema
   - Build and start the OHLC service

3. Test the streaming functionality using the stream client:

   ```bash
   go run cmd/client/stream_client.go
   ```

   The stream client will connect to the OHLC service and start receiving real-time candlestick data.

   And you will see the candlestick data being streamed in real-time.

```bash
   [ETHUSDT] 16:52:00 - Open: 1570.27, High: 1571.34, Low: 1569.35, Close: 1570.89, Volume: 123.90 (Period: 16:52:00 - 16:53:00)
   [BTCUSDT] 16:52:00 - Open: 83714.68, High: 83714.68, Low: 83674.00, Close: 83695.09, Volume: 5.01 (Period: 16:52:00 - 16:53:00)
   [BTCUSDT] 16:53:00 - Open: 83695.10, High: 83696.66, Low: 83664.04, Close: 83664.05, Volume: 6.37 (Period: 16:53:00 - 16:54:00) (Interval: 1m0s)
   [ETHUSDT] 16:53:00 - Open: 1570.89, High: 1571.50, Low: 1570.75, Close: 1570.82, Volume: 120.54 (Period: 16:53:00 - 16:54:00) (Interval: 1m0s)
```

### Troubleshooting

- If the service fails to connect to PostgreSQL, ensure the database is healthy:

  ```bash
  docker-compose ps
  ```

- Check service logs:

  ```bash
  docker-compose logs ohlc
  ```

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

## Deployment Guide

### 1. Build and Push Docker Image

1. Build the Docker image:

   ```bash
   docker build -t your-registry/ohlc:latest .
   ```

2. Push the image to your container registry:

   ```bash
   docker push your-registry/ohlc:latest
   ```

### 2. Configure DigitalOcean Access

1. Install doctl if not already installed:

   ```bash
   brew install doctl  # For macOS
   ```

2. Authenticate with DigitalOcean:

   ```bash
   doctl auth init
   ```

3. Export your DigitalOcean API token:

   ```bash
   export DO_TOKEN=your_digitalocean_api_token
   ```

### 3. Deploy with Terraform

1. Navigate to the Terraform directory:

   ```bash
   cd deployments/terraform
   ```

2. Initialize Terraform:

   ```bash
   terraform init
   ```

3. Review the deployment plan:

   ```bash
   terraform plan -var="do_token=$DO_TOKEN" \
     -var="image=your-registry/ohlc:latest" \
     -var="postgres_password=your-secure-password"
   ```

4. Apply the configuration:

   ```bash
   terraform apply -var="do_token=$DO_TOKEN" \
     -var="image=your-registry/ohlc:latest" \
     -var="postgres_password=your-secure-password"
   ```

### 4. Configure Kubernetes Access

1. Configure kubectl to use the new cluster:

   ```bash
   doctl kubernetes cluster kubeconfig save ohlc-cluster
   ```

2. Verify the deployment:

   ```bash
   kubectl get pods -n ohlc
   kubectl get services -n ohlc
   ```

### 5. Testing the Service

1. Port-forward the gRPC service:

   ```bash
   kubectl port-forward service/ohlc-grpc 50051:50051 -n ohlc
   ```

2. Use a gRPC client (like grpcurl) to test the service:

   ```bash
   grpcurl -plaintext localhost:50051 ohlc.OHLC/GetCandlesticks
   ```

### 6. Cleanup

To destroy the infrastructure when no longer needed:

```bash
terraform destroy -var="do_token=$DO_TOKEN"
```

## Configuration

The following variables can be customized in `terraform.tfvars` or via command line:

- `do_region`: DigitalOcean region (default: "nyc1")
- `do_k8s_version`: Kubernetes version (default: "1.28.2-do.0")
- `do_node_size`: Node size (default: "s-2vcpu-4gb")
- `replicas`: Number of service replicas (default: 1)
- `postgres_user`: PostgreSQL username (default: "ohlc")
- `postgres_db`: PostgreSQL database name (default: "ohlc")
- `grpc_port`: gRPC service port (default: 50051)

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
