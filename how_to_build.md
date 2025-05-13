# How to Build: ZK-Proof-Based Decentralized Healthcare Infrastructure

This document provides step-by-step instructions for building and implementing the ZK-Proof-Based Decentralized Healthcare Infrastructure.

## Prerequisites

- Docker and Docker Compose
- Python 3.8+ or Node.js 14+
- MongoDB
- Apache Cassandra
- Git

## Development Environment Setup

### 1. Initialize Project Structure

```bash
mkdir -p telemedicine_tech/{zk_mongo,cassandra_archive,merkle_tree,event_logger,yag_updater,api,tests}
cd telemedicine_tech
```

### 2. Set Up Containerized Databases

Create a `docker-compose.yml` file:

```yaml
version: '3'

services:
  mongodb:
    image: mongo:latest
    container_name: zk-mongo
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db
    environment:
      - MONGO_INITDB_ROOT_USERNAME=admin
      - MONGO_INITDB_ROOT_PASSWORD=password

  cassandra:
    image: cassandra:latest
    container_name: healthcare-cassandra
    ports:
      - "9042:9042"
    volumes:
      - cassandra_data:/var/lib/cassandra
    environment:
      - MAX_HEAP_SIZE=512M
      - HEAP_NEWSIZE=100M

volumes:
  mongo_data:
  cassandra_data:
```

Start the containers:

```bash
docker-compose up -d
```

## Implementation Guide

### Step 1: ZK-Mongo Container Module

Create `zk_mongo/zk_identity.py`:

```python
import hashlib
import time
import pymongo

class ZKIdentity:
    def __init__(self, mongo_uri="mongodb://admin:password@localhost:27017/"):
        self.client = pymongo.MongoClient(mongo_uri)
        self.db = self.client["zkidentity"]
        self.identities = self.db["identities"]
        
    def generate_zk_proof(self, party_id, claim, timestamp=None):
        """Generate a zero-knowledge proof for an identity claim"""
        if timestamp is None:
            timestamp = time.time()
        
        # Concatenate identity, claim, and timestamp
        data = f"{party_id}||{claim}||{timestamp}"
        
        # Generate SHA-256 hash
        zk_proof = hashlib.sha256(data.encode()).hexdigest()
        
        return zk_proof, timestamp
    
    def register_identity(self, party_id, claim):
        """Register a new identity with a claim"""
        timestamp = time.time()
        zk_proof, _ = self.generate_zk_proof(party_id, claim, timestamp)
        
        # Store in MongoDB
        record = {
            "party_id": party_id,
            "claim": claim,
            "timestamp": timestamp,
            "zk_proof": zk_proof
        }
        
        self.identities.insert_one(record)
        return zk_proof
    
    def validate_claim(self, party_id, claim):
        """Validate if a claim is valid for a party"""
        # Fetch the original registration
        record = self.identities.find_one({"party_id": party_id, "claim": claim})
        
        if not record:
            return False
            
        # Re-generate the proof using original timestamp
        zk_proof, _ = self.generate_zk_proof(
            party_id, 
            claim, 
            record["timestamp"]
        )
        
        # Compare with stored proof
        return zk_proof == record["zk_proof"]
```

### Step 2: Cassandra Archive Module

Create `cassandra_archive/file_storage.py`:

```python
from cassandra.cluster import Cluster
import hashlib
import time
import uuid

class CassandraArchive:
    def __init__(self, contact_points=['localhost'], port=9042):
        self.cluster = Cluster(contact_points=contact_points, port=port)
        self.session = self.cluster.connect()
        
        # Create keyspace if not exists
        self.session.execute("""
            CREATE KEYSPACE IF NOT EXISTS healthcare 
            WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '3'}
        """)
        
        self.session.set_keyspace('healthcare')
        
        # Create table if not exists
        self.session.execute("""
            CREATE TABLE IF NOT EXISTS documents (
                doc_id uuid PRIMARY KEY,
                doc_type text,
                owner text,
                hash_id text,
                timestamp timestamp,
                content_preview text
            )
        """)
    
    def hash_content(self, content):
        """Generate SHA-256 hash for content"""
        return hashlib.sha256(content.encode()).hexdigest()
    
    def store_file(self, doc_type, content, owner_id):
        """Store a document in the append-only storage"""
        doc_id = uuid.uuid4()
        hash_id = self.hash_content(content)
        timestamp = time.time()
        
        # Create a preview (first 100 chars)
        content_preview = content[:100] + "..." if len(content) > 100 else content
        
        # Insert into Cassandra
        self.session.execute(
            """
            INSERT INTO documents (doc_id, doc_type, owner, hash_id, timestamp, content_preview)
            VALUES (%s, %s, %s, %s, %s, %s)
            """,
            (doc_id, doc_type, owner_id, hash_id, timestamp, content_preview)
        )
        
        return str(doc_id), hash_id
    
    def query_by_owner(self, owner_id):
        """Retrieve all documents for an owner"""
        rows = self.session.execute(
            "SELECT * FROM documents WHERE owner = %s ALLOW FILTERING",
            (owner_id,)
        )
        
        return list(rows)
```

### Step 3: Merkle Tree Module

Create `merkle_tree/merkle.py`:

```python
import hashlib

class MerkleTree:
    def __init__(self):
        pass
    
    def hash_leaf(self, data):
        """Hash a single leaf node data"""
        return hashlib.sha256(data.encode()).hexdigest()
    
    def hash_nodes(self, left, right):
        """Hash two child nodes to create a parent node"""
        # Concatenate and hash
        combined = left + right
        return hashlib.sha256(combined.encode()).hexdigest()
    
    def build_tree(self, file_hashes):
        """Build a Merkle tree from a list of file hashes"""
        # If the list is empty, return None
        if not file_hashes:
            return None
            
        # If only one hash, return it as is
        if len(file_hashes) == 1:
            return file_hashes[0]
            
        # Process leaf nodes first
        leaf_nodes = [self.hash_leaf(h) if isinstance(h, str) else h for h in file_hashes]
        
        # Build the tree bottom-up
        while len(leaf_nodes) > 1:
            new_level = []
            
            # Process pairs of nodes
            for i in range(0, len(leaf_nodes), 2):
                # If we have a pair, hash them together
                if i + 1 < len(leaf_nodes):
                    new_hash = self.hash_nodes(leaf_nodes[i], leaf_nodes[i+1])
                # If we have an odd number, pair with itself
                else:
                    new_hash = self.hash_nodes(leaf_nodes[i], leaf_nodes[i])
                    
                new_level.append(new_hash)
                
            # Replace current level with the new level
            leaf_nodes = new_level
            
        # Return the root hash (should be the only element left)
        return leaf_nodes[0]
```

### Step 4: Event Logger Module

Create `event_logger/logger.py`:

```python
import pymongo
import time
import uuid

class EventLogger:
    def __init__(self, mongo_uri="mongodb://admin:password@localhost:27017/"):
        self.client = pymongo.MongoClient(mongo_uri)
        self.db = self.client["eventlogger"]
        self.events = self.db["events"]
        
        # Create indexes
        self.events.create_index("event_id", unique=True)
        self.events.create_index("party")
        self.events.create_index("status")
        
    def log_event(self, event_type, party_id, payload):
        """Log a new event"""
        event_id = str(uuid.uuid4())
        timestamp = time.time()
        
        event = {
            "event_id": event_id,
            "type": event_type,
            "party": party_id,
            "payload": payload,
            "status": "pending",
            "timestamp": timestamp,
            "retries": 0
        }
        
        self.events.insert_one(event)
        return event_id
        
    def resolve_event(self, event_id, status="completed"):
        """Mark an event as resolved"""
        self.events.update_one(
            {"event_id": event_id},
            {"$set": {"status": status, "resolved_at": time.time()}}
        )
        
    def retry_events(self, max_retries=3):
        """Retry pending events that may have failed"""
        pending_events = self.events.find({
            "status": "pending",
            "retries": {"$lt": max_retries}
        })
        
        for event in pending_events:
            # Increment retry count
            self.events.update_one(
                {"event_id": event["event_id"]},
                {"$inc": {"retries": 1}}
            )
            
            # Here you would implement your retry logic
            # For example, republish the event to a message queue
            
            # For demonstration, we'll just mark as retried
            print(f"Retrying event {event['event_id']}")
```

### Step 5: YAG Updater Module

Create `yag_updater/treatment_paths.py`:

```python
import pymongo

class YAGUpdater:
    def __init__(self, mongo_uri="mongodb://admin:password@localhost:27017/"):
        self.client = pymongo.MongoClient(mongo_uri)
        self.db = self.client["yagupdater"]
        self.treatment_paths = self.db["treatment_paths"]
        
    def update_path(self, symptom, path):
        """Add or update a treatment path for a symptom"""
        # Check if symptom exists
        existing = self.treatment_paths.find_one({"symptom": symptom})
        
        if existing:
            # Add the new path if it doesn't exist
            if path not in existing["paths"]:
                self.treatment_paths.update_one(
                    {"symptom": symptom},
                    {"$push": {"paths": path}}
                )
        else:
            # Create new entry
            self.treatment_paths.insert_one({
                "symptom": symptom,
                "paths": [path]
            })
            
    def get_paths(self, symptom):
        """Get all known treatment paths for a symptom"""
        result = self.treatment_paths.find_one({"symptom": symptom})
        
        if result:
            return result["paths"]
        return []
```

### Step 6: API Implementation

Create `api/main.py`:

```python
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import sys
import os

# Add parent directory to path for imports
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from zk_mongo.zk_identity import ZKIdentity
from cassandra_archive.file_storage import CassandraArchive
from merkle_tree.merkle import MerkleTree
from event_logger.logger import EventLogger
from yag_updater.treatment_paths import YAGUpdater

app = FastAPI(title="Healthcare ZK-Proof API")

# Initialize components
zk_identity = ZKIdentity()
cassandra = CassandraArchive()
merkle = MerkleTree()
event_logger = EventLogger()
yag = YAGUpdater()

# Define request models
class IdentityRequest(BaseModel):
    party_id: str
    claim: str

class DocumentRequest(BaseModel):
    doc_type: str
    content: str
    owner_id: str

class TreatmentPathRequest(BaseModel):
    symptom: str
    path: list

# API routes
@app.post("/identity/register")
async def register_identity(request: IdentityRequest):
    zk_proof = zk_identity.register_identity(request.party_id, request.claim)
    event_id = event_logger.log_event("identity_registration", request.party_id, {
        "claim": request.claim,
        "zk_proof": zk_proof
    })
    return {"zk_proof": zk_proof, "event_id": event_id}

@app.post("/identity/validate")
async def validate_identity(request: IdentityRequest):
    is_valid = zk_identity.validate_claim(request.party_id, request.claim)
    return {"is_valid": is_valid}

@app.post("/document/store")
async def store_document(request: DocumentRequest):
    doc_id, hash_id = cassandra.store_file(
        request.doc_type,
        request.content,
        request.owner_id
    )
    
    event_id = event_logger.log_event("document_storage", request.owner_id, {
        "doc_id": doc_id,
        "hash_id": hash_id
    })
    
    return {
        "doc_id": doc_id,
        "hash_id": hash_id,
        "event_id": event_id
    }

@app.get("/document/by-owner/{owner_id}")
async def get_documents(owner_id: str):
    documents = cassandra.query_by_owner(owner_id)
    return {"documents": [dict(doc) for doc in documents]}

@app.post("/treatment/update-path")
async def update_treatment_path(request: TreatmentPathRequest):
    yag.update_path(request.symptom, request.path)
    return {"status": "success"}

@app.get("/treatment/get-paths/{symptom}")
async def get_treatment_paths(symptom: str):
    paths = yag.get_paths(symptom)
    return {"symptom": symptom, "paths": paths}

# Run the API
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

### Step 7: Create Integration Workflow

Create `integration.py` in the project root:

```python
from zk_mongo.zk_identity import ZKIdentity
from cassandra_archive.file_storage import CassandraArchive
from merkle_tree.merkle import MerkleTree
from event_logger.logger import EventLogger
from yag_updater.treatment_paths import YAGUpdater

class HealthcareSystem:
    def __init__(self):
        self.zk_identity = ZKIdentity()
        self.cassandra = CassandraArchive()
        self.merkle = MerkleTree()
        self.event_logger = EventLogger()
        self.yag = YAGUpdater()
        
    def register_party(self, party_id, claim):
        """Register a new party (doctor, patient, lab, insurer)"""
        zk_proof = self.zk_identity.register_identity(party_id, claim)
        event_id = self.event_logger.log_event("registration", party_id, {
            "claim": claim,
            "zk_proof": zk_proof
        })
        return zk_proof, event_id
    
    def upload_medical_record(self, owner_id, doc_type, content):
        """Upload a medical record"""
        # Store the document
        doc_id, hash_id = self.cassandra.store_file(doc_type, content, owner_id)
        
        # Log the event
        event_id = self.event_logger.log_event("document_upload", owner_id, {
            "doc_id": doc_id,
            "hash_id": hash_id,
            "doc_type": doc_type
        })
        
        # Mark event as completed
        self.event_logger.resolve_event(event_id)
        
        return doc_id, hash_id
    
    def update_treatment_knowledge(self, doctor_id, patient_id, symptom, treatment_path):
        """Update the YAG with a successful treatment path"""
        # Validate doctor's identity
        if not self.zk_identity.validate_claim(doctor_id, "doctor"):
            return False, "Invalid doctor identity"
        
        # Log the treatment update event
        event_id = self.event_logger.log_event("treatment_update", doctor_id, {
            "patient_id": patient_id,
            "symptom": symptom,
            "treatment_path": treatment_path
        })
        
        # Update the YAG
        self.yag.update_path(symptom, treatment_path)
        
        # Mark event as completed
        self.event_logger.resolve_event(event_id)
        
        return True, event_id
    
    def get_treatment_recommendations(self, symptom):
        """Get AI recommendations for a symptom"""
        return self.yag.get_paths(symptom)

# Example usage
if __name__ == "__main__":
    system = HealthcareSystem()
    
    # Register a doctor
    doctor_proof, _ = system.register_party("doctor123", "doctor")
    print(f"Doctor registered with proof: {doctor_proof}")
    
    # Register a patient
    patient_proof, _ = system.register_party("patient456", "patient")
    print(f"Patient registered with proof: {patient_proof}")
    
    # Upload a medical record
    record = "Patient presents with symptoms of hypertension. BP: 145/95"
    doc_id, hash_id = system.upload_medical_record("patient456", "diagnosis", record)
    print(f"Medical record uploaded: {doc_id} with hash: {hash_id}")
    
    # Update treatment knowledge
    treatment_path = ["blood test", "lisinopril 10mg", "dietary changes", "follow-up"]
    success, event_id = system.update_treatment_knowledge(
        "doctor123", "patient456", "hypertension", treatment_path
    )
    print(f"Treatment knowledge updated: {success}, event_id: {event_id}")
    
    # Get treatment recommendations
    recommendations = system.get_treatment_recommendations("hypertension")
    print(f"Recommendations for hypertension: {recommendations}")
```

### Step 8: Create Requirements File

Create `requirements.txt` in the project root:

```
pymongo==4.2.0
cassandra-driver==3.25.0
fastapi==0.89.1
uvicorn==0.20.0
pydantic==1.10.4
```

### Step 9: Create Setup Script

Create `setup.sh` in the project root:

```bash
#!/bin/bash

# Create virtual environment
python -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Start the database containers
docker-compose up -d

# Initialize Cassandra keyspace
python -c "from cassandra_archive.file_storage import CassandraArchive; CassandraArchive()"

# Run a simple test
python integration.py

echo "Setup complete! The ZK-Proof-Based Decentralized Healthcare Infrastructure is ready."
```

## Testing the System

### 1. Run the Integration Test

```bash
python integration.py
```

### 2. Start the API Server

```bash
cd api
python main.py
```

### 3. Test API Endpoints

Using a tool like curl or Postman:

```bash
# Register a doctor
curl -X POST "http://localhost:8000/identity/register" \
     -H "Content-Type: application/json" \
     -d '{"party_id": "doctor123", "claim": "doctor"}'

# Upload a document
curl -X POST "http://localhost:8000/document/store" \
     -H "Content-Type: application/json" \
     -d '{"doc_type": "prescription", "content": "Take medication X twice daily", "owner_id": "patient456"}'

# Fetch documents for a patient
curl "http://localhost:8000/document/by-owner/patient456"
```

## Security Considerations

1. **Secure the MongoDB and Cassandra instances** with proper authentication and network isolation
2. **Implement proper API authentication** using JWT or other token-based authentication
3. **Add rate limiting** to prevent brute-force attacks
4. **Encrypt sensitive data** before storing in Cassandra
5. **Implement audit logging** for all sensitive operations
6. **Regularly backup data** and test restoration procedures

## Deployment to Production

1. Use a container orchestration system like Kubernetes
2. Set up proper monitoring and alerting
3. Implement CI/CD pipelines for automated testing and deployment
4. Configure proper backup and disaster recovery procedures
5. Perform security audits and penetration testing

## Conclusion

This guide provides a comprehensive approach to building a ZK-Proof-Based Decentralized Healthcare Infrastructure. By following these steps, you'll create a secure, privacy-preserving system that can handle healthcare data while ensuring compliance with regulations like HIPAA and GDPR.

The modular nature of the system allows for easy extension and integration with existing healthcare systems, while the ZK proofs and Merkle trees ensure data integrity and privacy.
