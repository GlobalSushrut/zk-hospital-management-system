// MongoDB initialization script for ZK Health Infrastructure

// Switch to admin db to create users
db = db.getSiblingDB('admin');

// Create application databases
db = db.getSiblingDB('zk_health');
db.createCollection('identity'); // For ZK identity module
db.createCollection('events');   // For event logging
db.createCollection('vectors');  // For treatment vectors
db.createCollection('consents'); // For consent management

// Create zkidentity database (used by the Go server)
db = db.getSiblingDB('zkidentity');
db.createCollection('identities'); // For ZK identity module

// Create indexes for identity collection
db.identity.createIndex({ "party_id": 1 }, { unique: true });
db.identity.createIndex({ "claim": 1 });

// Create indexes for events collection
db.events.createIndex({ "event_type": 1 });
db.events.createIndex({ "party_id": 1 });
db.events.createIndex({ "timestamp": 1 });

// Create indexes for vectors collection
db.vectors.createIndex({ "patient_id": 1 });
db.vectors.createIndex({ "doctor_id": 1 });
db.vectors.createIndex({ "status": 1 });

// Create indexes for consents collection
db.consents.createIndex({ "patient_id": 1 });
db.consents.createIndex({ "status": 1 });
db.consents.createIndex({ "expiry_date": 1 });

// Create application user
db = db.getSiblingDB('admin');
db.createUser({
  user: 'zk_health_user',
  pwd: 'zk_health_password',
  roles: [
    { role: 'readWrite', db: 'zk_health' },
    { role: 'readWrite', db: 'zkidentity' }
  ]
});
