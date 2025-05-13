# ZK Health - Hospital Management System Frontend

A comprehensive Hospital Management System that integrates with the ZK-Proof-Based Decentralized Healthcare Infrastructure, providing a secure and privacy-focused interface for healthcare providers.

## Features

- **Location-Based Policy Agreement Engine**: Enforces legal identity, compliance, and scope control based on geographical rules and roles
- **Comprehensive Patient Management**: Secure management of patient records with ZK proof verification 
- **Treatment Vector System**: Full treatment lifecycle management with plan creation, updates, and analytics
- **Oracle Agreement Integration**: Integration with the Oracle Chain Validator for legal and regulatory compliance
- **Document Management**: Secure uploading, verification, and retrieval of medical documents
- **Cross-Border Telemedicine**: Support for cross-jurisdictional healthcare delivery with proper validations

## Architecture

The system consists of:

1. **Frontend**: Python-based web application using FastAPI and Jinja2 templates
2. **Backend**: Go-based ZK-Proof Decentralized Healthcare Infrastructure
3. **Database**: MongoDB for storing Hospital Management System data

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Python 3.10+ (for development)
- Go 1.18+ (for backend development)

### Running with Docker

1. Clone the repository:

```bash
git clone https://github.com/your-org/telemedicine_tech.git
cd telemedicine_tech/frontend
```

2. Configure environment variables (optional - defaults are provided):

Create a `.env` file in the `frontend` directory with the following content:

```
DEBUG=true
HOST=0.0.0.0
PORT=8000
SECRET_KEY=your-secure-secret-key
ZK_API_BASE_URL=http://backend:8080
MONGODB_URL=mongodb://mongo:27017
MONGODB_DB=zk_health_hms
DEFAULT_COUNTRY=US
```

3. Start the application using Docker Compose:

```bash
docker-compose up -d
```

4. Access the application:

- Frontend: http://localhost:8000
- Backend API: http://localhost:8080

### Running Locally (Development)

1. Install dependencies:

```bash
pip install -r requirements.txt
```

2. Configure environment variables:

Create a `.env` file in the `frontend` directory with appropriate values, adjusting URL to point to your backend:

```
DEBUG=true
HOST=127.0.0.1
PORT=8000
SECRET_KEY=your-dev-secret-key
ZK_API_BASE_URL=http://localhost:8080
MONGODB_URL=mongodb://localhost:27017
MONGODB_DB=zk_health_hms
DEFAULT_COUNTRY=US
```

3. Run the application:

```bash
python main.py
```

4. Access the application at http://localhost:8000

## Integration with ZK Health Infrastructure

This Hospital Management System integrates with all components of the ZK Health Infrastructure:

- **Identity Management**: User registration, authentication, and ZK proof verification
- **Consent Management**: Creation and verification of patient consent for healthcare actions
- **Document Management**: Secure document storage with cryptographic verification
- **Treatment Vectors**: Management of patient treatment plans with privacy guarantees 
- **Policy Engine**: Location-based policy enforcement for healthcare actions
- **Oracle Chain**: Validation of healthcare actions against legal and regulatory requirements

## Security Features

- **Zero-Knowledge Proofs**: Privacy-preserving verification of identities and credentials
- **End-to-End Encryption**: Secure communication between frontend and backend
- **Role-Based Access Control**: Fine-grained permission management
- **Jurisdictional Validation**: Enforcement of healthcare regulations across borders
- **Comprehensive Audit Trails**: Immutable records of all actions and validations

## License

[Insert your license information here]
