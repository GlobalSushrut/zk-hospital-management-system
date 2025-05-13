"""
Demo module for ZK Health CLI
"""

import os
import time
import json
import random
import uuid
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.progress import track
from rich.markdown import Markdown

from utils import make_api_request

def run_demo(console):
    """
    Run an automated demo showcasing the ZK Health infrastructure capabilities
    """
    console.print(Panel.fit(
        "[bold blue]ZK-Proof Based Decentralized Healthcare Infrastructure[/bold blue]\n\n"
        "This demo will showcase the key components of the infrastructure, demonstrating\n"
        "how they work together to provide a secure, private, and compliant healthcare solution.",
        title="Interactive Demo", border_style="green"
    ))
    
    # Set up demo data
    demo_data = initialize_demo_data()
    
    # Display demo participants
    display_participants(console, demo_data)
    
    # Begin the demo scenario
    console.print("\n[bold yellow]Starting demo scenario: Cross-border telemedicine consultation[/bold yellow]")
    console.print("Press Enter to continue through each step...", style="dim")
    input()
    
    # Step 1: Register Identities with ZK Proofs
    step_1_register_identities(console, demo_data)
    input()
    
    # Step 2: Create Oracle Agreement for Cross-Border Telemedicine
    step_2_create_oracle_agreement(console, demo_data)
    input()
    
    # Step 3: Establish Patient Consent
    step_3_establish_consent(console, demo_data)
    input()
    
    # Step 4: Secure Medical Document Upload
    step_4_document_upload(console, demo_data)
    input()
    
    # Step 5: Start Treatment Vector with AI Recommendations
    step_5_treatment_vector(console, demo_data)
    input()
    
    # Step 6: API Gateway Token Generation and Access Control
    step_6_api_gateway(console, demo_data)
    input()
    
    # Demo complete
    console.print(Panel.fit(
        "[bold green]Demo completed successfully![/bold green]\n\n"
        "You've experienced the complete workflow of the ZK-Proof Based Decentralized\n"
        "Healthcare Infrastructure. This system provides:\n\n"
        "✅ Zero-knowledge identity verification\n"
        "✅ Regulatory compliance through Oracle Agreements\n"
        "✅ Patient-controlled consent management\n"
        "✅ Secure document storage with Merkle proofs\n"
        "✅ AI-assisted treatment recommendations\n"
        "✅ Secure API access with role-based permissions",
        title="Demo Complete", border_style="green"
    ))

def initialize_demo_data():
    """Initialize demo data for participants and entities"""
    return {
        "participants": {
            "doctor": {
                "id": f"doctor_{uuid.uuid4().hex[:8]}",
                "name": "Dr. Sarah Chen",
                "location": "Canada",
                "specialty": "Cardiology",
                "zk_proof": f"zkp_{uuid.uuid4().hex}"
            },
            "patient": {
                "id": f"patient_{uuid.uuid4().hex[:8]}",
                "name": "Raj Patel",
                "location": "India",
                "age": 45,
                "zk_proof": f"zkp_{uuid.uuid4().hex}"
            },
            "admin": {
                "id": f"admin_{uuid.uuid4().hex[:8]}",
                "name": "System Administrator",
                "zk_proof": f"zkp_{uuid.uuid4().hex}"
            }
        },
        "entities": {
            "hospital": {
                "id": f"hospital_{uuid.uuid4().hex[:8]}",
                "name": "Global Health Connect",
                "location": "Virtual"
            }
        },
        "generated": {
            "consent_id": None,
            "agreement_id": None,
            "document_id": None,
            "vector_id": None,
            "token_id": None
        }
    }

def display_participants(console, demo_data):
    """Display demo participants"""
    console.print("\n[bold]Demo Participants:[/bold]")
    
    table = Table(show_header=True, header_style="bold blue")
    table.add_column("Role")
    table.add_column("Name")
    table.add_column("ID")
    table.add_column("Location")
    
    doctor = demo_data["participants"]["doctor"]
    patient = demo_data["participants"]["patient"]
    admin = demo_data["participants"]["admin"]
    hospital = demo_data["entities"]["hospital"]
    
    table.add_row(
        "Doctor",
        doctor["name"],
        doctor["id"],
        doctor["location"]
    )
    
    table.add_row(
        "Patient",
        patient["name"],
        patient["id"],
        patient["location"]
    )
    
    table.add_row(
        "Admin",
        admin["name"],
        admin["id"],
        "Global"
    )
    
    table.add_row(
        "Hospital",
        hospital["name"],
        hospital["id"],
        hospital["location"]
    )
    
    console.print(table)

def step_1_register_identities(console, demo_data):
    """Demo Step 1: Register participants with ZK proofs"""
    console.print(Panel.fit(
        "[bold]Step 1: Zero-Knowledge Identity Registration[/bold]\n\n"
        "In this step, participants register their identities using zero-knowledge proofs.\n"
        "These proofs allow verification of identity claims without exposing actual personal data.",
        title="Identity Management", border_style="blue"
    ))
    
    doctor = demo_data["participants"]["doctor"]
    patient = demo_data["participants"]["patient"]
    admin = demo_data["participants"]["admin"]
    
    # Simulate registering doctor
    console.print(f"\n[bold]Registering doctor:[/bold] {doctor['name']}")
    for _ in track(range(5), description="Generating ZK proof..."):
        time.sleep(0.3)
    
    console.print(f"[green]✓ Doctor registered successfully![/green]")
    console.print(f"  ID: {doctor['id']}")
    console.print(f"  Claim: doctor")
    console.print(f"  ZK Proof: {doctor['zk_proof'][:10]}...{doctor['zk_proof'][-5:]}")
    
    # Simulate registering patient
    console.print(f"\n[bold]Registering patient:[/bold] {patient['name']}")
    for _ in track(range(5), description="Generating ZK proof..."):
        time.sleep(0.3)
    
    console.print(f"[green]✓ Patient registered successfully![/green]")
    console.print(f"  ID: {patient['id']}")
    console.print(f"  Claim: patient")
    console.print(f"  ZK Proof: {patient['zk_proof'][:10]}...{patient['zk_proof'][-5:]}")
    
    # Simulate registering admin
    console.print(f"\n[bold]Registering admin:[/bold] {admin['name']}")
    for _ in track(range(5), description="Generating ZK proof..."):
        time.sleep(0.3)
    
    console.print(f"[green]✓ Admin registered successfully![/green]")
    console.print(f"  ID: {admin['id']}")
    console.print(f"  Claim: admin")
    console.print(f"  ZK Proof: {admin['zk_proof'][:10]}...{admin['zk_proof'][-5:]}")

def step_2_create_oracle_agreement(console, demo_data):
    """Demo Step 2: Create Oracle Agreement for Cross-Border Telemedicine"""
    console.print(Panel.fit(
        "[bold]Step 2: Oracle Agreement for Cross-Border Telemedicine[/bold]\n\n"
        "This step creates an Oracle Agreement that automatically enforces compliance\n"
        "with telemedicine regulations spanning both Canada and India.",
        title="Oracle Chain Validator", border_style="blue"
    ))
    
    admin = demo_data["participants"]["admin"]
    
    # Display agreement clauses
    agreement_clauses = [
        {
            "clause_id": "india-telemedicine-jurisdiction",
            "title": "India Telemedicine Guidelines Compliance",
            "description": "Enforces compliance with Telemedicine Practice Guidelines by Medical Council of India",
            "preconditions": {
                "doctor_licensed": True,
                "patient_identity_verified": True,
                "consent_obtained": True
            }
        },
        {
            "clause_id": "canada-telemedicine-jurisdiction",
            "title": "Canadian Virtual Care Requirements",
            "description": "Enforces compliance with Canadian provincial telemedicine regulations",
            "preconditions": {
                "doctor_registered_in_province": True,
                "patient_informed_consent": True,
                "secure_communication": True
            }
        },
        {
            "clause_id": "cross-border-data-transfer",
            "title": "Cross-Border PHI Protection",
            "description": "Ensures Protected Health Information compliance across borders",
            "preconditions": {
                "data_minimization": True,
                "encryption_in_transit": True,
                "patient_consent_for_transfer": True
            }
        }
    ]
    
    console.print("\n[bold]Creating Oracle Agreement with the following clauses:[/bold]")
    
    table = Table(show_header=True, header_style="bold blue")
    table.add_column("Clause ID")
    table.add_column("Title")
    table.add_column("Description")
    
    for clause in agreement_clauses:
        table.add_row(
            clause["clause_id"],
            clause["title"],
            clause["description"]
        )
    
    console.print(table)
    
    # Simulate creating agreement
    console.print(f"\n[bold]Admin {admin['name']} creating agreement...[/bold]")
    for _ in track(range(8), description="Processing agreement..."):
        time.sleep(0.4)
    
    # Generate agreement ID
    agreement_id = f"agreement_{uuid.uuid4().hex[:10]}"
    demo_data["generated"]["agreement_id"] = agreement_id
    
    console.print(f"[green]✓ Oracle agreement created successfully![/green]")
    console.print(f"  Agreement ID: {agreement_id}")
    console.print(f"  Jurisdiction: INDIA-CANADA-TELEMEDICINE")
    console.print(f"  Clauses: {len(agreement_clauses)}")

def step_3_establish_consent(console, demo_data):
    """Demo Step 3: Establish Patient Consent"""
    console.print(Panel.fit(
        "[bold]Step 3: Multi-Party Consent Management[/bold]\n\n"
        "This step establishes patient consent for treatment and data sharing.\n"
        "The consent agreement specifies exactly what data can be shared, with whom,\n"
        "and for what purpose - all cryptographically verifiable.",
        title="Consent Management", border_style="blue"
    ))
    
    doctor = demo_data["participants"]["doctor"]
    patient = demo_data["participants"]["patient"]
    hospital = demo_data["entities"]["hospital"]
    
    # Simulate creating consent
    console.print(f"\n[bold]Creating consent agreement for telemedicine consultation[/bold]")
    console.print(f"Patient: {patient['name']} ({patient['id']})")
    console.print(f"Primary physician: {doctor['name']} ({doctor['id']})")
    console.print(f"Institution: {hospital['name']} ({hospital['id']})")
    
    consent_details = {
        "type": "treatment",
        "description": "Telemedicine cardiology consultation and data sharing",
        "expiry_days": 30,
        "all_required": True,
        "resources": [
            "medical_history",
            "diagnostic_reports",
            "prescription_data"
        ]
    }
    
    console.print("\n[bold]Consent details:[/bold]")
    console.print(f"  Type: {consent_details['type']}")
    console.print(f"  Description: {consent_details['description']}")
    console.print(f"  Duration: {consent_details['expiry_days']} days")
    console.print(f"  All parties required: {consent_details['all_required']}")
    console.print("  Resources:")
    for resource in consent_details['resources']:
        console.print(f"    - {resource}")
    
    for _ in track(range(6), description="Creating consent agreement..."):
        time.sleep(0.4)
    
    # Generate consent ID
    consent_id = f"consent_{uuid.uuid4().hex[:10]}"
    demo_data["generated"]["consent_id"] = consent_id
    
    console.print(f"\n[green]✓ Consent agreement created with ID: {consent_id}[/green]")
    
    # Simulate patient approval
    console.print(f"\n[bold]Patient {patient['name']} approving consent...[/bold]")
    for _ in track(range(3), description="Verifying patient identity..."):
        time.sleep(0.3)
    
    console.print(f"[green]✓ Patient approval recorded with ZK proof[/green]")
    
    # Simulate doctor approval
    console.print(f"\n[bold]Doctor {doctor['name']} approving consent...[/bold]")
    for _ in track(range(3), description="Verifying doctor identity..."):
        time.sleep(0.3)
    
    console.print(f"[green]✓ Doctor approval recorded with ZK proof[/green]")
    console.print(f"[green bold]✓ Consent is now ACTIVE[/green bold]")

def step_4_document_upload(console, demo_data):
    """Demo Step 4: Secure Document Upload"""
    console.print(Panel.fit(
        "[bold]Step 4: Tamper-Proof Document Management[/bold]\n\n"
        "Medical documents are uploaded to the Cassandra Archive with Merkle tree verification.\n"
        "This ensures documents cannot be modified without detection and creates\n"
        "a verifiable audit trail for all access.",
        title="Cassandra Archive", border_style="blue"
    ))
    
    patient = demo_data["participants"]["patient"]
    doctor = demo_data["participants"]["doctor"]
    consent_id = demo_data["generated"]["consent_id"]
    
    # Simulate document upload
    document_types = ["lab_result", "medical_record", "prescription"]
    selected_type = random.choice(document_types)
    
    file_name = {
        "lab_result": "ecg_results.pdf",
        "medical_record": "patient_history.pdf",
        "prescription": "heart_medication.pdf"
    }[selected_type]
    
    file_size = random.randint(150, 500)
    
    console.print(f"\n[bold]Uploading document:[/bold] {file_name}")
    console.print(f"Type: {selected_type}")
    console.print(f"Size: {file_size} KB")
    console.print(f"Patient: {patient['name']} ({patient['id']})")
    console.print(f"Uploader: {doctor['name']} ({doctor['id']})")
    console.print(f"Consent ID: {consent_id}")
    
    for _ in track(range(10), description="Uploading and generating Merkle proof..."):
        time.sleep(0.3)
    
    # Generate document ID and hashes
    document_id = f"doc_{uuid.uuid4().hex[:10]}"
    file_hash = f"hash_{uuid.uuid4().hex}"
    merkle_root = f"merkle_{uuid.uuid4().hex}"
    demo_data["generated"]["document_id"] = document_id
    
    console.print(f"\n[green]✓ Document uploaded successfully![/green]")
    console.print(f"  Document ID: {document_id}")
    console.print(f"  File Hash: {file_hash[:10]}...{file_hash[-5:]}")
    console.print(f"  Merkle Root: {merkle_root[:10]}...{merkle_root[-5:]}")
    
    # Simulate document verification
    console.print(f"\n[bold]Verifying document integrity...[/bold]")
    for _ in track(range(5), description="Verifying Merkle proof..."):
        time.sleep(0.2)
    
    console.print(f"[green]✓ Document integrity verified![/green]")
    console.print(f"  The document has not been tampered with")
    console.print(f"  Verification can be repeated by any authorized party")

def step_5_treatment_vector(console, demo_data):
    """Demo Step 5: Treatment Vector with AI Recommendations"""
    console.print(Panel.fit(
        "[bold]Step 5: AI-Assisted Treatment Pathways[/bold]\n\n"
        "The YAG AI engine provides treatment recommendations based on the patient's\n"
        "condition and medical history. The Misalignment Tracker monitors deviations\n"
        "from recommended treatment paths and flags potential issues.",
        title="YAG AI & Treatment Vectors", border_style="blue"
    ))
    
    patient = demo_data["participants"]["patient"]
    doctor = demo_data["participants"]["doctor"]
    
    # Simulate starting treatment vector
    symptom = "Chest pain with arrhythmia"
    
    console.print(f"\n[bold]Starting treatment vector[/bold]")
    console.print(f"Patient: {patient['name']} ({patient['id']})")
    console.print(f"Doctor: {doctor['name']} ({doctor['id']})")
    console.print(f"Primary symptom: {symptom}")
    
    for _ in track(range(7), description="Analyzing medical history and generating recommendations..."):
        time.sleep(0.4)
    
    # Generate vector ID
    vector_id = f"vector_{uuid.uuid4().hex[:10]}"
    demo_data["generated"]["vector_id"] = vector_id
    
    console.print(f"\n[green]✓ Treatment vector started successfully![/green]")
    console.print(f"  Vector ID: {vector_id}")
    
    # Display AI recommendations
    recommended_path = [
        "ECG and blood work panel",
        "Holter monitoring for 24 hours",
        "Beta blocker medication (low dose)",
        "Diet and lifestyle modifications",
        "Follow-up in 2 weeks"
    ]
    
    console.print("\n[bold]YAG AI Recommended Treatment Path:[/bold]")
    for i, step in enumerate(recommended_path, 1):
        console.print(f"  {i}. {step}")
    
    # Simulate updating treatment
    console.print(f"\n[bold]Doctor updating treatment vector...[/bold]")
    action = "Prescribed beta blocker and calcium channel blocker"
    
    for _ in track(range(4), description="Processing update..."):
        time.sleep(0.3)
    
    console.print(f"\n[yellow]⚠ Misalignment detected with recommended path![/yellow]")
    console.print(f"  Action: {action}")
    console.print(f"  Misalignment: Addition of calcium channel blocker not in recommended path")
    console.print(f"  Misalignment score: 0.35 (Low risk)")
    console.print(f"  Recommendation: Document reason for deviation in notes")
    
    # Simulate doctor adding notes
    console.print(f"\n[bold]Doctor adding justification notes...[/bold]")
    notes = "Patient has family history of similar condition that responded well to dual therapy."
    
    for _ in track(range(3), description="Updating records..."):
        time.sleep(0.3)
    
    console.print(f"\n[green]✓ Treatment updated with justification[/green]")
    console.print(f"  Notes: {notes}")

def step_6_api_gateway(console, demo_data):
    """Demo Step 6: API Gateway Token Generation and Access Control"""
    console.print(Panel.fit(
        "[bold]Step 6: Secure API Access Control[/bold]\n\n"
        "The ZK API Gateway provides secure access to the healthcare infrastructure\n"
        "through token-based authentication and role-based access control.\n"
        "Rate limiting prevents abuse and ensures system availability.",
        title="ZK API Gateway", border_style="blue"
    ))
    
    doctor = demo_data["participants"]["doctor"]
    
    # Simulate token generation
    console.print(f"\n[bold]Generating API token for {doctor['name']}[/bold]")
    console.print(f"Party ID: {doctor['id']}")
    console.print(f"Claim: doctor")
    console.print(f"Validity: 24 hours")
    
    for _ in track(range(5), description="Generating secure token..."):
        time.sleep(0.4)
    
    # Generate token ID
    token_id = f"token_{uuid.uuid4().hex}"
    demo_data["generated"]["token_id"] = token_id
    
    console.print(f"\n[green]✓ ZK API token generated successfully![/green]")
    console.print(f"  Token ID: {token_id[:15]}...{token_id[-5:]}")
    console.print(f"  Party ID: {doctor['id']}")
    console.print(f"  Claim: doctor")
    console.print(f"  Valid for: 24 hours")
    
    # Show rate limits
    console.print("\n[bold]Active rate limits for doctor role:[/bold]")
    
    rate_limit_table = Table(show_header=True, header_style="bold blue")
    rate_limit_table.add_column("Endpoint")
    rate_limit_table.add_column("Per Minute")
    rate_limit_table.add_column("Per Hour")
    rate_limit_table.add_column("Per Day")
    
    rate_limit_table.add_row(
        "/api/*",
        "60",
        "300",
        "1000"
    )
    
    rate_limit_table.add_row(
        "/api/treatment/*",
        "30",
        "150",
        "500"
    )
    
    rate_limit_table.add_row(
        "/api/document/*",
        "20",
        "100",
        "200"
    )
    
    console.print(rate_limit_table)
    
    # Show access control in action
    console.print("\n[bold]Testing access control with generated token:[/bold]")
    
    access_table = Table(show_header=True, header_style="bold blue")
    access_table.add_column("Endpoint")
    access_table.add_column("Access")
    access_table.add_column("Reason")
    
    access_table.add_row(
        "/api/patient/history",
        "[green]GRANTED[/green]",
        "Doctor role has access to patient history"
    )
    
    access_table.add_row(
        "/api/treatment/update",
        "[green]GRANTED[/green]",
        "Doctor role can update treatments"
    )
    
    access_table.add_row(
        "/api/admin/users",
        "[red]DENIED[/red]",
        "Doctor role cannot access admin endpoints"
    )
    
    console.print(access_table)
    
    # Show token usage
    console.print("\n[bold]Using token for API access:[/bold]")
    console.print("Include the following header in API requests:")
    console.print(f"[blue]X-ZK-API-Key: {token_id[:15]}...{token_id[-5:]}[/blue]")
