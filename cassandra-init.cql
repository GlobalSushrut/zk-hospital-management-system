-- Create zkhealth keyspace
CREATE KEYSPACE IF NOT EXISTS zkhealth 
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

-- Create document archive table
CREATE TABLE IF NOT EXISTS zkhealth.document_archive (
    document_id text,
    version int,
    content_hash text,
    created_at timestamp,
    metadata map<text, text>,
    PRIMARY KEY (document_id, version)
);

-- Create event log table
CREATE TABLE IF NOT EXISTS zkhealth.event_log (
    event_id text,
    event_type text,
    party_id text,
    timestamp timestamp,
    payload text,
    PRIMARY KEY (event_id)
);

-- Create indexes for quicker lookups
CREATE INDEX IF NOT EXISTS ON zkhealth.event_log (event_type);
CREATE INDEX IF NOT EXISTS ON zkhealth.event_log (party_id);
CREATE INDEX IF NOT EXISTS ON zkhealth.event_log (timestamp);

-- Create verified proofs table
CREATE TABLE IF NOT EXISTS zkhealth.verified_proofs (
    proof_id text,
    party_id text,
    claim_type text,
    timestamp timestamp,
    proof_data text,
    verification_result boolean,
    PRIMARY KEY (proof_id)
);

CREATE INDEX IF NOT EXISTS ON zkhealth.verified_proofs (party_id);
