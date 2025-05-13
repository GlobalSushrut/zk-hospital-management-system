# ZK-Proof Healthcare System - Applications

## Overview

The ZK-Proof Healthcare System includes several applications that demonstrate different aspects of healthcare data management with enhanced privacy and security through zero-knowledge proofs. This document provides an overview of these applications, their purpose, and how they leverage the infrastructure components.

## Core Applications

### 1. Healthcare Workflow Demo

**Location:** `/cmd/healthcare_demo/`

**Purpose:** Demonstrates a real-world healthcare workflow with actual HTTP requests to the infrastructure APIs, validating patient consent, cross-jurisdiction data access, role-based access control, emergency access, and document storage.

**Key Features:**
- Creation of zero-knowledge proofs for patient consent
- Cross-jurisdiction medical record sharing with policy validation
- Role-based access control for different healthcare roles
- Emergency access override with proper security controls
- Secure document storage and retrieval

**Usage:**
```bash
./bin/healthcare_demo
```

### 2. Infrastructure Validation Tool

**Location:** `/cmd/workflow_demo/`

**Purpose:** Validates the functionality of all infrastructure components in a simulated healthcare environment, ensuring that all services work correctly together.

**Key Features:**
- Comprehensive testing of ZK circuit execution and verification
- Validation of policy enforcement for various healthcare scenarios
- Simulation of different healthcare roles and access patterns
- Verification of emergency access protocols
- Complete end-to-end healthcare workflow simulation

**Usage:**
```bash
./bin/workflow_demo [--policy-only] [--timeout <seconds>]
```

### 3. Policy Validation Server

**Location:** `/pkg/policy/`

**Purpose:** Provides a standalone server for healthcare data access policy validation, enforcement of role-based access, cross-jurisdiction sharing agreements, and data sensitivity handling.

**Key Features:**
- Basic policy validation for healthcare data access
- Role-based policy validation for different healthcare professionals
- Cross-jurisdiction validation for multi-region healthcare providers
- Emergency access override validation
- Document storage with proper access controls

**Usage:**
```bash
# The policy server is started automatically by healthcare_demo or workflow_demo
# It can also be accessed directly at http://localhost:8081/
```

### 4. Infrastructure Validation Script

**Location:** `/cli/validate_infrastructure.py`

**Purpose:** Provides a comprehensive validation suite for the ZK-Proof Healthcare System's infrastructure components through a series of HTTP tests.

**Key Features:**
- Health checks for all infrastructure components
- ZK circuit execution and verification testing
- Scaling and load balancing validation
- Security token generation and verification
- Monitoring and health checking
- Interoperability with FHIR and EHR systems
- Policy enforcement validation
- Load testing for performance validation

**Usage:**
```bash
python3 cli/validate_infrastructure.py [--skip-load-test]
```

## Benchmark Applications

### 1. Infrastructure Benchmark Tool

**Location:** `/cli/benchmark_infrastructure.py`

**Purpose:** Measures the performance of the ZK-Proof Healthcare System's infrastructure components, focusing on throughput, latency, and resource utilization.

**Key Features:**
- Performance measurement of ZK proof generation and verification
- Policy validation throughput and latency measurement
- Scaling capabilities under different load conditions
- Load testing with configurable parameters
- Comprehensive reporting of benchmark results

**Usage:**
```bash
python3 cli/benchmark_infrastructure.py [--duration <seconds>] [--threads <count>]
```

### 2. Treatment Benchmark Tool

**Location:** `/cli/benchmark_treatment.py`

**Purpose:** Specifically measures the performance of treatment-related workflows, including patient record access, validation of treatment protocols, and consent verification.

**Key Features:**
- Simulation of treatment workflows in high-throughput environments
- Measurement of end-to-end latency for treatment validation
- Testing of concurrent patient treatment scenarios
- Validation of security and privacy guarantees under load

**Usage:**
```bash
python3 cli/benchmark_treatment.py [--patients <count>] [--doctors <count>]
```

## Administrative Applications

### 1. Infrastructure Manager

**Location:** `/pkg/infrastructure/`

**Purpose:** Core management component that orchestrates all infrastructure services, provides a REST API for component interaction, and manages configuration.

**Key Features:**
- Centralized management of all infrastructure components
- Dynamic scaling of compute resources based on demand
- Monitoring and health checking of all services
- Security management including token generation and verification
- ZK circuit template management and execution
- Integration with healthcare interoperability standards

**Access:**
- REST API available at http://localhost:8080/
- Health check endpoint: http://localhost:8080/health
- Full API documentation in `/docs/api.md`

## Integration Components

### 1. EHR Integration Client

**Location:** `/pkg/interop/ehr.go`

**Purpose:** Provides integration with Electronic Health Record (EHR) systems, allowing secure access to patient records while maintaining privacy.

**Key Features:**
- Authentication with various EHR systems
- Retrieval of patient records with proper authorization
- Verification of data access policies
- Support for emergency access scenarios
- Audit logging of all access operations

### 2. FHIR Client

**Location:** `/pkg/interop/fhir.go`

**Purpose:** Implements FHIR (Fast Healthcare Interoperability Resources) standard for healthcare data exchange, enabling interoperability with other healthcare systems.

**Key Features:**
- Full support for FHIR R4 resources
- Patient resource creation and retrieval
- Observation and condition management
- Secure transmission of healthcare data
- Privacy-preserving data exchange with ZK proofs

## Development Tools

### 1. Zero-Knowledge Circuit Builder

**Location:** `/pkg/zkcircuit/`

**Purpose:** Provides tools for building, testing, and deploying zero-knowledge circuits for healthcare use cases.

**Key Features:**
- Circuit templates for common healthcare privacy scenarios
- Compiler for zero-knowledge proofs
- Testing framework for circuit validation
- Performance optimization tools
- Integration with the broader infrastructure

## System Requirements

- **Operating System:** Linux (Ubuntu 20.04 LTS or newer recommended)
- **CPU:** 4+ cores recommended for production use
- **Memory:** 8GB+ RAM recommended
- **Storage:** 50GB+ available disk space
- **Network:** Stable internet connection for FHIR and EHR integration
- **Software Dependencies:**
  - Go 1.16 or newer
  - Python 3.8 or newer
  - MongoDB 4.4 or newer
  - Cassandra 4.0 or newer (for archival storage)

## Getting Started

1. Clone the repository
2. Build the applications:
   ```bash
   go build -o ./bin/healthcare_demo ./cmd/healthcare_demo
   go build -o ./bin/workflow_demo ./cmd/workflow_demo
   ```
3. Run the healthcare workflow demo:
   ```bash
   ./bin/healthcare_demo
   ```
4. Run the infrastructure validation:
   ```bash
   python3 cli/validate_infrastructure.py
   ```

For detailed documentation on each application, see the respective README files in each component directory.
