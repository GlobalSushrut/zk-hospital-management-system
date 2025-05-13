# ZK-Proof Based Hospital Management System - User Guide

## Introduction

Welcome to the ZK-Proof Based Hospital Management System! This guide provides step-by-step instructions for using the system's key features. Whether you're a healthcare provider, administrator, or IT professional, this guide will help you navigate the system efficiently.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Identity Management](#identity-management)
3. [Document Management](#document-management)
4. [Policy Management](#policy-management)
5. [API Gateway](#api-gateway)
6. [Benchmarking](#benchmarking)
7. [Troubleshooting](#troubleshooting)
8. [FAQ](#faq)

## Getting Started

### System Requirements

- Modern web browser (Chrome, Firefox, Safari, Edge)
- Network connectivity to the system server
- Authentication credentials for your role

### Logging In

1. Navigate to the system URL: `https://your-hospital-domain.com/login`
2. Enter your credentials (username and password)
3. Complete any two-factor authentication if required
4. You will be directed to the dashboard appropriate for your role

### Navigation Basics

The main dashboard includes:
- Left sidebar: Main navigation menu
- Top bar: Quick actions, notifications, and user profile
- Main content area: Context-specific information and tools
- Footer: System version, support links, and legal information

## Identity Management

### Registering a New Identity

1. Navigate to **Identity → Register New**
2. Complete the registration form with:
   - Party ID (auto-generated or custom)
   - Claim type (e.g., patient, doctor, admin)
   - Supporting information
3. Click **Generate ZK Proof**
4. Review the generated proof information
5. Click **Submit** to complete registration

### Validating an Identity

1. Navigate to **Identity → Validate**
2. Enter the party ID or scan the QR code
3. Select the claim to validate
4. Click **Verify**
5. The system will display the verification result without revealing sensitive data

### Retrieving Identity Information

1. Navigate to **Identity → Retrieve**
2. Enter the party ID to look up
3. Click **Search**
4. Review the available identity information (filtered by your access level)

## Document Management

### Uploading a Document

1. Navigate to **Documents → Upload**
2. Complete the document form:
   - Document name
   - Document type
   - Owner ID
   - Tags (optional)
3. Attach the document file (PDF, DICOM, HL7, etc.)
4. Click **Upload**
5. The system will display a confirmation with the document ID

### Retrieving Documents

1. Navigate to **Documents → Retrieve**
2. Search by:
   - Owner ID
   - Document ID (if known)
   - Document type
   - Date range
3. Click **Search**
4. Browse the results list
5. Click any document to view details or download (if authorized)

### Verifying a Document

1. Navigate to **Documents → Verify**
2. Enter the document ID
3. Upload the document to verify or provide the content hash
4. Click **Verify**
5. The system will display the verification result, including:
   - Original hash match
   - Timestamp validation
   - Chain of custody information

## Policy Management

### Checking Policy Compliance

1. Navigate to **Policy → Validate**
2. Select the policy type:
   - Data access
   - Cross-jurisdiction
   - Role-based
3. Enter the context information:
   - Resource ID
   - Requester ID
   - Action type
4. Click **Check Compliance**
5. Review the detailed compliance report

### Role Validation

1. Navigate to **Policy → Role Validation**
2. Enter the user ID
3. Select the role to validate
4. Click **Validate**
5. The system will confirm if the user has the specified role privileges

### Policy Updates

1. Navigate to **Policy → Oracle Integration**
2. Click **Check for Updates**
3. Review any new policy guidelines
4. Click **Apply Updates** to implement new policies
5. The system will confirm successful policy updates

## API Gateway

### Generating Access Tokens

1. Navigate to **Gateway → Token Management**
2. Select **Generate New Token**
3. Configure token parameters:
   - Expiration time
   - Scope
   - Resource limitations
4. Click **Generate Token**
5. Copy and securely store the generated token

### Managing API Routes

1. Navigate to **Gateway → Route Management**
2. Browse existing routes or click **Add New Route**
3. Configure route parameters:
   - Path pattern
   - Target service
   - Authentication requirements
   - Rate limits
4. Click **Save Route**
5. Test the route by sending a sample request

## Benchmarking

The system includes comprehensive benchmarking tools to evaluate performance. These tools are primarily for system administrators and technical staff.

### Running Full Benchmarks

1. Access the server command line
2. Navigate to the project directory
3. Run the benchmark command:
   ```bash
   cd /path/to/telemedicine_tech
   python cli/benchmark.py benchmark --iterations 100
   ```
4. Review the generated reports in the console output
5. Optional: Save detailed results to a file:
   ```bash
   python cli/benchmark.py benchmark --iterations 100 --output benchmark_results.json
   ```

### Running Targeted Benchmarks

1. Access the server command line
2. Navigate to the project directory
3. Run the specific benchmark component:
   ```bash
   # For identity benchmarks only
   python cli/benchmark_identity.py
   
   # For document benchmarks only
   python cli/benchmark_document.py
   
   # For policy benchmarks only
   python cli/benchmark_policy.py
   
   # For gateway benchmarks only
   python cli/benchmark_gateway.py
   ```
4. Review the component-specific performance metrics

## Troubleshooting

### Common Issues and Solutions

#### Identity Verification Failures

**Problem**: Identity verification returns "Invalid proof"
**Solution**: 
1. Ensure the party ID is correctly entered
2. Check that the claim type matches the original registration
3. Verify the ZK proof hasn't expired
4. Try regenerating the proof

#### Document Retrieval Issues

**Problem**: Document retrieval returns "No documents found"
**Solution**:
1. Verify the owner ID is correct
2. Check access permissions for the requested documents
3. Ensure documents exist for the specified criteria
4. Check for formatting issues in the owner ID (try removing hyphens)

#### API Connection Problems

**Problem**: API returns connection timeouts
**Solution**:
1. Verify the server is running (`curl http://localhost:8080/health`)
2. Check network connectivity to the server
3. Ensure the API gateway is properly configured
4. Verify authentication tokens haven't expired

#### Performance Degradation

**Problem**: System response times are slow
**Solution**:
1. Run benchmarks to identify bottlenecks
2. Check Cassandra connection and performance
3. Verify adequate system resources (CPU, memory)
4. Review recent changes that might impact performance
5. Check for excessive load or DDoS attacks

## FAQ

### General Questions

**Q: What is a zero-knowledge proof?**

A: A zero-knowledge proof is a cryptographic method where one party (the prover) can prove to another party (the verifier) that they know a specific piece of information without revealing the information itself. In our system, this allows identity verification without exposing sensitive personal data.

**Q: How secure is the document storage?**

A: Documents are secured through multiple layers:
1. Content hashing to verify integrity
2. Encryption during transit and at rest
3. Owner-based access controls
4. Audit logging of all access attempts
5. Zero-knowledge proofs for sensitive operations

**Q: Can the system work with our existing EHR?**

A: Yes, the system provides integration APIs compatible with major EHR systems including Epic, Cerner, and Allscripts. Custom integrations can be developed for other systems.

### Technical Questions

**Q: What database does the system use?**

A: The system uses Apache Cassandra as its primary database, configured for optimal performance in healthcare scenarios with appropriate consistency levels and replication factors.

**Q: How does the caching system work?**

A: The system implements an LRU (Least Recently Used) caching mechanism for identities and documents, optimizing retrieval performance while maintaining memory efficiency. Cache statistics are monitored to ensure optimal hit rates.

**Q: What is the recommended deployment architecture?**

A: For production environments, we recommend:
1. At least 3 application servers behind a load balancer
2. A Cassandra cluster with at least 3 nodes
3. Geographical distribution for disaster recovery
4. Regular backup and restore testing

### Support Resources

- **Documentation**: Complete system documentation at `/docs`
- **API Reference**: Interactive API documentation at `/api/docs`
- **Support Portal**: Submit and track support tickets at `support.your-hospital-domain.com`
- **Training Videos**: Video tutorials at `training.your-hospital-domain.com`

For urgent assistance, contact technical support at support@telemedicine-tech.com or call 1-800-TELE-SUPPORT.
