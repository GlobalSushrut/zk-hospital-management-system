# ZK-Proof Based Hospital Management System - Build Guide

## What We've Built

The ZK-Proof Based Hospital Management System is a comprehensive healthcare platform that leverages zero-knowledge proofs to protect patient privacy while enabling secure sharing of medical information. The system combines state-of-the-art cryptography with efficient database design to create a fast, reliable platform for healthcare providers.

## Key Components

### 1. Backend Services (Go)

We've developed a robust Go-based backend that provides:

- RESTful API endpoints for all system functionality
- Zero-knowledge proof generation and verification
- Document management with content integrity verification
- Policy enforcement and compliance checking
- Identity management with privacy protections

### 2. Database Layer (Cassandra)

The system uses Apache Cassandra for data persistence with the following features:

- Optimized for high-throughput operations
- Configured with appropriate consistency levels (ONE for development)
- Document storage with efficient indexing
- Identity record storage with cryptographic protections

### 3. Benchmarking Tools (Python)

We've created comprehensive benchmarking tools that:

- Test all API endpoints under varying load conditions
- Measure performance metrics (response time, throughput)
- Validate functionality against specifications
- Provide detailed reports on system performance

## Build Instructions

### Prerequisites

- Go 1.21+
- Python 3.8+
- Apache Cassandra 4.0+
- Docker (optional, for containerized deployment)
- Git

### Step 1: Clone the Repository

```bash
git clone https://github.com/your-org/telemedicine-tech.git
cd telemedicine-tech
```

### Step 2: Set Up Cassandra

```bash
# Start Cassandra (use Docker for simplicity)
docker run -p 9042:9042 --name cassandra-db -d cassandra:latest

# Wait for Cassandra to initialize
sleep 30

# Create keyspace and tables
cqlsh -f scripts/setup_cassandra.cql
```

### Step 3: Build the Go Server

```bash
cd cmd/server
go build -o telemedicine-server main.go
```

### Step 4: Run the Server

```bash
./telemedicine-server --port 8080 --cassandra-host localhost
```

### Step 5: Set Up the Python Environment

```bash
cd ../../cli
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### Step 6: Run Benchmarks

```bash
python benchmark.py benchmark --iterations 100
```

## Configuration Options

The system can be configured through environment variables or a configuration file:

### Environment Variables

- `PORT`: Server port (default: 8080)
- `CASSANDRA_HOSTS`: Comma-separated list of Cassandra hosts
- `CASSANDRA_KEYSPACE`: Keyspace name (default: "telemedicine")
- `CONSISTENCY_LEVEL`: Cassandra consistency level (default: "ONE")
- `LOG_LEVEL`: Logging verbosity (default: "info")
- `ENABLE_DEBUG`: Enable debug endpoints (default: false)

### Configuration File (config.yaml)

```yaml
server:
  port: 8080
  debug: false
  log_level: info

database:
  type: cassandra
  hosts:
    - localhost
  keyspace: telemedicine
  consistency: ONE
  replication_factor: 1

security:
  jwt_secret: your-secret-key
  token_expiry: 24h
  min_password_length: 10
```

## Build Customization

### Production Build

For a production-ready build with optimizations:

```bash
go build -tags prod -ldflags="-s -w" -o telemedicine-server main.go
```

### Development Build

For a development build with additional debug information:

```bash
go build -tags dev,debug -o telemedicine-server main.go
```

### Cross-Compilation

To build for different platforms:

```bash
# For Linux
GOOS=linux GOARCH=amd64 go build -o telemedicine-server-linux main.go

# For Windows
GOOS=windows GOARCH=amd64 go build -o telemedicine-server.exe main.go

# For macOS
GOOS=darwin GOARCH=amd64 go build -o telemedicine-server-mac main.go
```

## Monitoring the Build

The system includes built-in telemetry endpoints:

- `/health`: Basic health check
- `/metrics`: Prometheus-compatible metrics
- `/debug/pprof`: Performance profiling (development only)

## Continuous Integration

Our CI pipeline automatically:

1. Runs unit and integration tests
2. Performs static code analysis
3. Builds binaries for multiple platforms
4. Conducts performance benchmarks
5. Deploys to staging environment for verification
