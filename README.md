# OHLC Trading Chart Service

## Project Structure

```markdown
.
├── cmd/                    # Application entry points
│   └── ohlc/              # Main service executable
├── internal/              # Private application code
│   ├── binance/           # Binance API client
│   ├── candlestick/       # OHLC data processing
│   ├── storage/           # Data persistence layer
│   └── streaming/         # gRPC streaming service
├── pkg/                   # Public libraries
├── proto/                 # Protocol buffer definitions
├── deployments/           # Kubernetes and deployment configs
│   ├── helm/              # Helm charts
│   └── terraform/         # Terraform configurations
└── test/                  # Integration and e2e tests
```

## Prerequisites

- Go 1.21 or later
- Docker
- Terraform
- DigitalOcean account and API token
- Protocol Buffers compiler
- SQLite3
- kubectl
- doctl (DigitalOcean CLI)

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
