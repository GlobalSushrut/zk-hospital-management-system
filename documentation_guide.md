# ZK-Proof Based Hospital Management System - Documentation

## Overview

This document provides comprehensive technical documentation for the ZK-Proof Based Hospital Management System. It covers the system architecture, key components, API specifications, and integration details.

## System Architecture

The Hospital Management System is built on a microservices architecture with the following components:

1. **Core API Server (Go)**
   - REST API endpoints for all system functionality
   - Built with Go and Gorilla Mux router
   - Handles authentication, authorization, and request routing

2. **Cassandra Database**
   - Document storage and retrieval
   - Identity management data
   - Configured with ONE consistency level for optimal performance
   - Uses replication factor of 1 for development, 3 for production

3. **ZK-Proof Engine**
   - Zero-knowledge proof generation and verification
   - Identity validation without exposing sensitive data
   - Claim-based access control

4. **Policy Service**
   - RBAC (Role-Based Access Control)
   - Cross-jurisdiction compliance
   - Regulatory validation
   - Policy Oracle integration

5. **Document Management System**
   - Secure document storage with content hashing
   - Document verification and validation
   - Owner-based document retrieval
   - ZK-proof based document access control

## API Specifications

### Identity Management API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/identity/register` | POST | Register a new identity with ZK proof |
| `/identity/validate` | POST | Validate an identity using ZK proof |
| `/identity/retrieve/{id}` | GET | Retrieve identity by ID |
| `/identity/{id}` | GET | Alternative endpoint for identity retrieval |

### Document Management API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/document/store` | POST | Upload and store a new document |
| `/document/verify` | POST | Verify document authenticity |
| `/document/by-owner/{owner}` | GET | Retrieve documents by owner ID |
| `/document/zk-proof` | POST | Generate ZK proof for document access |

### Policy API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/policy/validate` | POST | Validate against policy rules |
| `/policy/cross-jurisdiction` | POST | Check cross-jurisdiction compliance |
| `/policy/role-validation` | POST | Validate user roles against policies |
| `/policy/validator` | GET | Get validator information |
| `/policy/oracle` | POST | Check policy oracle for regulatory updates |

### Gateway API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/gateway/token/generate` | POST | Generate authentication token |
| `/gateway/token/validate` | POST | Validate authentication token |
| `/gateway/route` | GET | Route request to appropriate service |
| `/gateway/throttle` | GET | Apply rate limiting to requests |
| `/gateway/rbac` | POST | Check RBAC permissions |
| `/gateway/cross-service` | POST | Authenticate across services |

## Database Schema

### Identity Collection

```
CREATE TABLE identities (
  party_id UUID PRIMARY KEY,
  claim TEXT,
  zk_proof TEXT,
  created_at TIMESTAMP,
  metadata MAP<TEXT, TEXT>
);
```

### Documents Collection

```
CREATE TABLE documents (
  doc_id UUID PRIMARY KEY,
  doc_type TEXT,
  owner TEXT,
  hash_id UUID,
  timestamp TIMESTAMP,
  content_preview TEXT,
  content_hash TEXT
);

CREATE INDEX ON documents (owner);
```

## Performance Specifications

Based on comprehensive benchmarks, the system performs with the following metrics:

- ZK Proof Generation: ~6ms average response time, 166 ops/sec
- Identity Verification: ~4.3ms average response time, 230 ops/sec
- Document Upload: ~6.1ms average response time, 163 ops/sec
- Policy Validation: ~2.9ms average response time, 339 ops/sec
- Request Throttling: ~2.5ms average response time, 404 ops/sec

## Security Considerations

1. **Zero-Knowledge Proofs**
   - Patient identity is protected using ZK proofs
   - Medical data can be verified without exposing sensitive information
   - Claims can be validated without revealing underlying data

2. **Role-Based Access Control**
   - Fine-grained permission system
   - Role hierarchy for medical staff
   - Jurisdiction-based access restrictions

3. **Document Security**
   - Content hashing for integrity verification
   - Owner-based access control
   - Audit trail for all document operations

## Integration Points

The system provides the following integration points:

1. **REST API**
   - Primary integration method for all clients
   - JSON-based request/response format
   - Bearer token authentication

2. **Event Stream**
   - Real-time updates for critical events
   - Webhook notifications for subscribers
   - Message queues for asynchronous processing

3. **Health Monitoring**
   - Prometheus metrics endpoint
   - Grafana dashboard integration
   - Health check endpoint for load balancers

## Deployment Requirements

- Go 1.21+
- Cassandra 4.0+
- 4 CPU cores minimum
- 8GB RAM minimum
- 50GB storage minimum
- HTTPS with TLS 1.3
- Load balancer for production deployments
