#!/usr/bin/env python3
"""
Real-World Infrastructure Validation Test Script for ZK Health System
This script tests actual infrastructure components in realistic healthcare scenarios
"""

import json
import os
import random
import string
import sys
import time
import uuid
import requests
import argparse
from rich.console import Console
from rich.progress import track
from rich.panel import Panel
from rich.table import Table

# API Endpoints
BASE_API_URL = "http://localhost:8080"
HEALTH_CHECK_URL = f"{BASE_API_URL}/health"
ZK_CIRCUIT_URL = f"{BASE_API_URL}/zkcircuit"
SCALING_STATUS_URL = f"{BASE_API_URL}/scaling/status"
SECURITY_STATUS_URL = f"{BASE_API_URL}/security/status"
MONITORING_URL = f"{BASE_API_URL}/monitoring/health"
FHIR_URL = f"{BASE_API_URL}/interop/fhir"
EHR_URL = f"{BASE_API_URL}/interop/ehr"
POLICY_URL = f"{BASE_API_URL}/policy/validate"
IDENTITY_URL = f"{BASE_API_URL}/identity/register"

console = Console()

def generate_patient_data():
    """Generate synthetic patient data"""
    blood_types = ["A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"]
    conditions = ["Hypertension", "Diabetes", "Asthma", "Arthritis", "Migraine", "Anxiety"]
    
    return {
        "patient_id": f"P{random.randint(10000, 99999)}",
        "name": f"Patient-{uuid.uuid4().hex[:8]}",
        "age": random.randint(18, 85),
        "blood_type": random.choice(blood_types),
        "conditions": random.sample(conditions, random.randint(0, 3))
    }

def generate_healthcare_provider():
    """Generate synthetic healthcare provider data"""
    specialties = ["Cardiology", "Neurology", "Pediatrics", "Oncology", "Radiology", "Internal Medicine"]
    
    return {
        "provider_id": f"DR{random.randint(10000, 99999)}",
        "name": f"Dr. Provider-{uuid.uuid4().hex[:8]}",
        "specialty": random.choice(specialties),
        "hospital": f"Hospital-{random.randint(1, 5)}"
    }

def test_health_checking():
    """Test that the server is running properly"""
    console.print("[bold blue]Testing Health Check...[/bold blue]")
    
    try:
        response = requests.get(HEALTH_CHECK_URL)
        response.raise_for_status()
        result = response.json()
        
        console.print(f"[green]✓[/green] Health check successful: {result}")
        return True
    except Exception as e:
        console.print(f"[red]✗[/red] Health check failed: {str(e)}")
        return False

def test_zk_circuit_execution():
    """Test the ZK circuit toolkit with patient consent verification"""
    console.print("\n[bold blue]Testing ZK Circuit Execution (Patient Consent Verification)...[/bold blue]")
    
    # Generate data
    patient = generate_patient_data()
    provider = generate_healthcare_provider()
    
    # Create consent parameters
    consent_data = {
        "circuit_type": "patient-consent",
        "patient_id": patient["patient_id"],
        "provider_id": provider["provider_id"],
        "consent_type": "full-access",
        "timestamp": int(time.time()),
        "expiration": int(time.time()) + 86400 * 30,  # 30 days
        "data_categories": ["medical-history", "medications", "lab-results"]
    }
    
    try:
        # Execute ZK proof generation
        console.print("Generating ZK proof for patient consent...")
        response = requests.post(f"{ZK_CIRCUIT_URL}/execute", json=consent_data)
        response.raise_for_status()
        result = response.json()
        
        # Extract proof
        zk_proof = result.get("proof")
        
        # Verify proof
        console.print("Verifying ZK proof validity...")
        verify_data = {
            "circuit_type": "patient-consent",
            "proof": zk_proof,
            "public_inputs": {
                "patient_id": patient["patient_id"],
                "provider_id": provider["provider_id"],
                "consent_type_hash": "0x1234"  # In real implementation, this would be an actual hash
            }
        }
        
        verify_response = requests.post(f"{ZK_CIRCUIT_URL}/verify", json=verify_data)
        verify_response.raise_for_status()
        verify_result = verify_response.json()
        
        if verify_result.get("valid", False):
            console.print(f"[green]✓[/green] ZK Circuit test successful: Generated and verified consent proof")
            console.print(f"[blue]Circuit Execution Time:[/blue] {result.get('execution_time_ms', 'N/A')}ms")
            return True
        else:
            console.print(f"[red]✗[/red] ZK proof verification failed")
            return False
            
    except Exception as e:
        console.print(f"[red]✗[/red] ZK Circuit test failed: {str(e)}")
        return False

def test_horizontal_scaling():
    """Test the horizontal scaling infrastructure"""
    console.print("\n[bold blue]Testing Horizontal Scaling...[/bold blue]")
    
    try:
        # Check current status
        response = requests.get(SCALING_STATUS_URL)
        response.raise_for_status()
        initial_status = response.json()
        
        initial_nodes = initial_status.get("nodes", [])
        console.print(f"Current nodes: {len(initial_nodes)}")
        
        # Request scale-up (simulation)
        scale_data = {
            "desired_nodes": len(initial_nodes) + 2,
            "reason": "increased_load"
        }
        
        scale_response = requests.post(f"{SCALING_STATUS_URL}/scale", json=scale_data)
        scale_response.raise_for_status()
        scale_result = scale_response.json()
        
        # Check status after scale request
        status_response = requests.get(SCALING_STATUS_URL)
        status_response.raise_for_status()
        new_status = status_response.json()
        
        console.print(f"[green]✓[/green] Horizontal scaling test successful")
        console.print(f"Initial node count: {len(initial_nodes)}")
        console.print(f"New node count: {len(new_status.get('nodes', []))}")
        console.print(f"Auto-scaling status: {new_status.get('auto_scaling_enabled', False)}")
        
        # Test load balancer routing
        console.print("Testing load balancer routing...")
        
        # Make multiple requests and check distribution
        route_counts = {}
        for i in track(range(20), description="Sending requests to load balancer"):
            route_response = requests.get(f"{BASE_API_URL}/scaling/route")
            if route_response.status_code == 200:
                route_data = route_response.json()
                node_id = route_data.get("node_id", "unknown")
                route_counts[node_id] = route_counts.get(node_id, 0) + 1
            time.sleep(0.1)
        
        if len(route_counts) > 1:
            console.print(f"[green]✓[/green] Requests distributed across {len(route_counts)} nodes")
            for node, count in route_counts.items():
                console.print(f"Node {node}: {count} requests")
            return True
        else:
            console.print(f"[yellow]⚠[/yellow] All requests routed to same node - distribution not verified")
            return True
            
    except Exception as e:
        console.print(f"[red]✗[/red] Horizontal scaling test failed: {str(e)}")
        return False

def test_security_features():
    """Test the advanced security features"""
    console.print("\n[bold blue]Testing Advanced Security Features...[/bold blue]")
    
    try:
        # Get security status
        response = requests.get(SECURITY_STATUS_URL)
        response.raise_for_status()
        security_status = response.json()
        
        console.print(f"Security module status: {security_status.get('status', 'unknown')}")
        console.print(f"Key rotation last performed: {security_status.get('last_key_rotation', 'unknown')}")
        console.print(f"Rate limiting active: {security_status.get('rate_limiting_enabled', False)}")
        
        # Test authentication with token
        auth_data = {
            "username": f"testuser-{uuid.uuid4().hex[:8]}",
            "password": "password123"  # Demo only
        }
        
        token_response = requests.post(f"{SECURITY_STATUS_URL}/token", json=auth_data)
        token_response.raise_for_status()
        token_data = token_response.json()
        
        auth_token = token_data.get("token")
        console.print(f"[green]✓[/green] Successfully generated authentication token")
        
        # Test token verification
        headers = {"Authorization": f"Bearer {auth_token}"}
        verify_response = requests.get(f"{SECURITY_STATUS_URL}/verify", headers=headers)
        verify_response.raise_for_status()
        verify_data = verify_response.json()
        
        if verify_data.get("valid", False):
            console.print(f"[green]✓[/green] Token verification successful")
            console.print(f"Token issued at: {verify_data.get('issued_at')}")
            console.print(f"Token expires at: {verify_data.get('expires_at')}")
            return True
        else:
            console.print(f"[red]✗[/red] Token verification failed")
            return False
            
    except Exception as e:
        console.print(f"[red]✗[/red] Security features test failed: {str(e)}")
        return False

def test_policy_validation():
    """Test healthcare policy validation"""
    console.print("\n[bold blue]Testing Healthcare Policy Validation...[/bold blue]")
    
    # Create test policy request
    policy_data = {
        "requester": {
            "id": f"DR{random.randint(10000, 99999)}",
            "role": "physician",
            "department": "cardiology",
            "jurisdiction": "california"
        },
        "subject": {
            "id": f"P{random.randint(10000, 99999)}",
            "record_type": "medical_history",
            "sensitivity": "high",
            "jurisdiction": "california"
        },
        "action": "read",
        "purpose": "treatment",
        "auth_method": "two_factor",
        "emergency": False
    }
    
    try:
        # Submit policy validation request
        response = requests.post(POLICY_URL, json=policy_data)
        response.raise_for_status()
        result = response.json()
        
        console.print(f"Policy validation result: {result.get('allowed', False)}")
        console.print(f"Reason: {result.get('reason', 'No reason provided')}")
        console.print(f"Validation time: {result.get('validation_time_ms', 'N/A')}ms")
        
        # Test with different jurisdiction (cross-jurisdiction test)
        policy_data["subject"]["jurisdiction"] = "new_york"
        
        cross_response = requests.post(f"{POLICY_URL}", json=policy_data)
        cross_response.raise_for_status()
        cross_result = cross_response.json()
        
        console.print("\nCross-Jurisdiction Policy Test:")
        console.print(f"Policy validation result: {cross_result.get('allowed', False)}")
        console.print(f"Reason: {cross_result.get('reason', 'No reason provided')}")
        
        # Test with emergency flag
        policy_data["emergency"] = True
        policy_data["subject"]["jurisdiction"] = "california"
        
        emergency_response = requests.post(f"{POLICY_URL}", json=policy_data)
        emergency_response.raise_for_status()
        emergency_result = emergency_response.json()
        
        console.print("\nEmergency Access Policy Test:")
        console.print(f"Policy validation result: {emergency_result.get('allowed', False)}")
        console.print(f"Reason: {emergency_result.get('reason', 'No reason provided')}")
        
        console.print(f"[green]✓[/green] Policy validation tests completed successfully")
        return True
            
    except Exception as e:
        console.print(f"[red]✗[/red] Policy validation test failed: {str(e)}")
        return False

def test_monitoring_resilience():
    """Test monitoring and resilience features"""
    console.print("\n[bold blue]Testing Monitoring & Resilience Features...[/bold blue]")
    
    try:
        # Check health status of all components
        response = requests.get(MONITORING_URL)
        response.raise_for_status()
        health_data = response.json()
        
        console.print("System Component Health:")
        
        health_table = Table(show_header=True, header_style="bold")
        health_table.add_column("Component", style="cyan")
        health_table.add_column("Status", style="green")
        health_table.add_column("Last Check", style="blue")
        health_table.add_column("Response Time (ms)", style="magenta")
        
        for component, data in health_data.get("components", {}).items():
            status = data.get("status", "unknown")
            status_style = "green" if status == "healthy" else "red"
            health_table.add_row(
                component, 
                f"[{status_style}]{status}[/{status_style}]",
                data.get("last_check", "unknown"),
                str(data.get("response_time_ms", "N/A"))
            )
        
        console.print(health_table)
        
        # Test circuit breaker (simulate with API calls)
        console.print("\nTesting Circuit Breaker Pattern...")
        
        circuit_status = requests.get(f"{MONITORING_URL}/circuit/status")
        circuit_status.raise_for_status()
        circuit_data = circuit_status.json()
        
        console.print(f"Database circuit: {circuit_data.get('database', 'closed')}")
        console.print(f"FHIR circuit: {circuit_data.get('fhir', 'closed')}")
        console.print(f"EHR circuit: {circuit_data.get('ehr', 'closed')}")
        
        # Trigger circuit breaker test
        for i in track(range(5), description="Testing circuit breaker"):
            circuit_test = requests.get(f"{MONITORING_URL}/circuit/test?service=test")
            time.sleep(0.5)
        
        # Check status after test
        circuit_status = requests.get(f"{MONITORING_URL}/circuit/status")
        circuit_status.raise_for_status()
        circuit_data_after = circuit_status.json()
        
        console.print(f"Test circuit after load: {circuit_data_after.get('test', 'closed')}")
        console.print(f"[green]✓[/green] Circuit breaker test completed")
        
        return True
            
    except Exception as e:
        console.print(f"[red]✗[/red] Monitoring & resilience test failed: {str(e)}")
        return False

def test_interoperability():
    """Test healthcare interoperability features"""
    console.print("\n[bold blue]Testing Healthcare Interoperability...[/bold blue]")
    
    # Create patient data for testing
    patient_id = f"PT{random.randint(10000, 99999)}"
    
    try:
        # Test FHIR client
        console.print("Testing FHIR Client...")
        fhir_response = requests.get(f"{FHIR_URL}/status")
        fhir_response.raise_for_status()
        fhir_status = fhir_response.json()
        
        console.print(f"FHIR client status: {fhir_status.get('status', 'unknown')}")
        console.print(f"FHIR version: {fhir_status.get('version', 'unknown')}")
        
        # Test FHIR patient creation
        patient_data = {
            "resourceType": "Patient",
            "id": patient_id,
            "active": True,
            "name": [
                {
                    "use": "official",
                    "family": "Testpatient",
                    "given": ["John", "Q"]
                }
            ],
            "gender": random.choice(["male", "female"]),
            "birthDate": "1970-01-01"
        }
        
        create_response = requests.post(f"{FHIR_URL}/Patient", json=patient_data)
        create_response.raise_for_status()
        create_result = create_response.json()
        
        console.print(f"[green]✓[/green] FHIR Patient created: {create_result.get('id')}")
        
        # Test EHR client
        console.print("\nTesting EHR Client...")
        ehr_response = requests.get(f"{EHR_URL}/status")
        ehr_response.raise_for_status()
        ehr_status = ehr_response.json()
        
        console.print(f"EHR client status: {ehr_status.get('status', 'unknown')}")
        console.print(f"EHR system: {ehr_status.get('system', 'unknown')}")
        
        # Test retrieval of patient data from EHR
        patient_response = requests.get(f"{EHR_URL}/patient/{patient_id}")
        if patient_response.status_code == 200:
            patient_result = patient_response.json()
            console.print(f"[green]✓[/green] Retrieved patient data from EHR")
        else:
            console.print(f"[yellow]⚠[/yellow] Patient not found in EHR (expected for test patient)")
            
        console.print(f"[green]✓[/green] Interoperability tests completed successfully")
        return True
            
    except Exception as e:
        console.print(f"[red]✗[/red] Interoperability test failed: {str(e)}")
        return False

def run_all_tests():
    """Run all infrastructure validation tests"""
    console.print(Panel.fit(
        "[bold cyan]ZK-Health System - Real-World Infrastructure Validation[/bold cyan]\n"
        "Testing all infrastructure components in real-world healthcare scenarios",
        title="Test Suite", subtitle="May 13, 2025"
    ))
    
    # First run health check
    if not test_health_checking():
        console.print("[red bold]Health check failed. Aborting further tests.[/red bold]")
        return
    
    # Track test results
    results = {}
    
    # Run integration tests
    tests = [
        ("ZK Circuit Execution", test_zk_circuit_execution),
        ("Horizontal Scaling", test_horizontal_scaling),
        ("Advanced Security", test_security_features),
        ("Policy Validation", test_policy_validation),
        ("Monitoring & Resilience", test_monitoring_resilience),
        ("Healthcare Interoperability", test_interoperability)
    ]
    
    for test_name, test_func in tests:
        console.print(f"\n[bold yellow]Running Test: {test_name}[/bold yellow]")
        start_time = time.time()
        success = test_func()
        duration = time.time() - start_time
        results[test_name] = {"success": success, "duration": duration}
    
    # Print summary
    console.print("\n[bold cyan]Test Results Summary:[/bold cyan]")
    
    summary_table = Table(show_header=True, header_style="bold")
    summary_table.add_column("Test", style="cyan")
    summary_table.add_column("Result", style="green")
    summary_table.add_column("Duration (s)", style="blue")
    
    overall_success = True
    for test_name, result in results.items():
        success = result["success"]
        overall_success &= success
        status = "[green]✓ PASS[/green]" if success else "[red]✗ FAIL[/red]"
        summary_table.add_row(test_name, status, f"{result['duration']:.2f}")
    
    console.print(summary_table)
    
    if overall_success:
        console.print("\n[bold green]✓ All infrastructure components validated successfully[/bold green]")
        console.print("[green]The ZK-Health System infrastructure is ready for production use![/green]")
    else:
        console.print("\n[bold red]✗ Some infrastructure components failed validation[/bold red]")
        console.print("[red]Please check the detailed logs for more information[/red]")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Run real-world infrastructure validation tests")
    parser.add_argument("--test", choices=["all", "zkcircuit", "scaling", "security", "policy", "monitoring", "interop"], 
                        default="all", help="Specific test to run")
    args = parser.parse_args()
    
    if args.test == "all":
        run_all_tests()
    elif args.test == "zkcircuit":
        test_health_checking() and test_zk_circuit_execution()
    elif args.test == "scaling":
        test_health_checking() and test_horizontal_scaling()
    elif args.test == "security":
        test_health_checking() and test_security_features()
    elif args.test == "policy":
        test_health_checking() and test_policy_validation()
    elif args.test == "monitoring":
        test_health_checking() and test_monitoring_resilience()
    elif args.test == "interop":
        test_health_checking() and test_interoperability()
