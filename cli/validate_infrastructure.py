#!/usr/bin/env python3
"""
Infrastructure Validation Tool for ZK-Proof Healthcare System

This tool runs a series of real-world tests against the system infrastructure
to validate that all components are working correctly in a production-ready configuration.
"""

import argparse
import json
import logging
import random
import requests
import sys
import time
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timedelta

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('infrastructure_validation.log')
    ]
)
logger = logging.getLogger(__name__)

# Infrastructure API endpoints
BASE_URL = "http://localhost:8080"
HEALTH_ENDPOINT = f"{BASE_URL}/health"
ZK_EXECUTE_ENDPOINT = f"{BASE_URL}/zkcircuit/execute"
ZK_VERIFY_ENDPOINT = f"{BASE_URL}/zkcircuit/verify"
ZK_LIST_ENDPOINT = f"{BASE_URL}/zkcircuit/list"
SCALING_STATUS_ENDPOINT = f"{BASE_URL}/scaling/status"
SCALING_NODES_ENDPOINT = f"{BASE_URL}/scaling/nodes"
SECURITY_TOKEN_ENDPOINT = f"{BASE_URL}/security/token"
SECURITY_VERIFY_ENDPOINT = f"{BASE_URL}/security/verify"
MONITORING_HEALTH_ENDPOINT = f"{BASE_URL}/monitoring/health"
MONITORING_METRICS_ENDPOINT = f"{BASE_URL}/monitoring/metrics"
FHIR_STATUS_ENDPOINT = f"{BASE_URL}/interop/fhir/status"
EHR_STATUS_ENDPOINT = f"{BASE_URL}/interop/ehr/status"

# Policy server endpoints
POLICY_URL = "http://localhost:8081"
POLICY_VALIDATE_ENDPOINT = f"{POLICY_URL}/policy/validate"
POLICY_ROLE_ENDPOINT = f"{POLICY_URL}/policy/role"
POLICY_CROSS_JURIS_ENDPOINT = f"{POLICY_URL}/policy/cross-jurisdiction"

class ValidationContext:
    """Context object to track validation state and results"""
    
    def __init__(self):
        self.test_results = {}
        self.system_info = {}
        self.tokens = {}
        self.start_time = datetime.now()
        self.total_tests = 0
        self.passed_tests = 0
        self.failed_tests = 0
        self.skipped_tests = 0
        
    def add_result(self, test_name, passed, details=None, response=None):
        """Record a test result"""
        status = "PASS" if passed else "FAIL"
        self.test_results[test_name] = {
            "status": status,
            "details": details,
            "timestamp": datetime.now().isoformat(),
        }
        
        if response:
            self.test_results[test_name]["response"] = response
            
        self.total_tests += 1
        if passed:
            self.passed_tests += 1
            logger.info(f"✅ {test_name}: {status}")
        else:
            self.failed_tests += 1
            logger.error(f"❌ {test_name}: {status} - {details}")
    
    def add_system_info(self, key, value):
        """Add system information"""
        self.system_info[key] = value
    
    def get_summary(self):
        """Generate a summary of all test results"""
        duration = datetime.now() - self.start_time
        return {
            "summary": {
                "total_tests": self.total_tests,
                "passed_tests": self.passed_tests,
                "failed_tests": self.failed_tests,
                "skipped_tests": self.skipped_tests,
                "success_rate": f"{(self.passed_tests / self.total_tests) * 100:.1f}%" if self.total_tests > 0 else "N/A",
                "execution_time": str(duration),
                "timestamp": datetime.now().isoformat(),
            },
            "system_info": self.system_info,
            "results": self.test_results
        }

def http_get(url, headers=None):
    """Make HTTP GET request with error handling"""
    try:
        response = requests.get(url, headers=headers, timeout=10)
        return response
    except requests.exceptions.RequestException as e:
        logger.error(f"Error making GET request to {url}: {e}")
        return None

def http_post(url, payload, headers=None):
    """Make HTTP POST request with error handling"""
    try:
        if headers is None:
            headers = {"Content-Type": "application/json"}
        response = requests.post(url, json=payload, headers=headers, timeout=10)
        return response
    except requests.exceptions.RequestException as e:
        logger.error(f"Error making POST request to {url}: {e}")
        return None

def validate_infrastructure_health(context):
    """Validate basic health of infrastructure components"""
    logger.info("Validating infrastructure health...")
    
    # Check infrastructure API health
    response = http_get(HEALTH_ENDPOINT)
    if response and response.status_code == 200:
        health_data = response.json()
        context.add_result("Infrastructure Health Check", True, 
                          f"API reports healthy status", health_data)
        context.add_system_info("api_status", "healthy")
    else:
        context.add_result("Infrastructure Health Check", False, 
                          f"API health check failed. Status code: {response.status_code if response else 'No response'}")
        context.add_system_info("api_status", "unhealthy")

def validate_zk_circuit_operations(context):
    """Validate ZK circuit execution and verification"""
    logger.info("Validating ZK circuit operations...")
    
    # 1. Check available circuits
    response = http_get(ZK_LIST_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("ZK Circuit List", False, 
                          f"Failed to list ZK circuits. Status code: {response.status_code if response else 'No response'}")
        return
        
    circuits = response.json()
    context.add_result("ZK Circuit List", True, 
                      f"Found {len(circuits)} available circuits", circuits)
    
    # 2. Test patient consent circuit execution
    payload = {
        "circuit_type": "patient-consent",
        "public_inputs": {
            "patient_id": f"P{random.randint(1000, 9999)}",
            "provider_id": f"D{random.randint(1000, 9999)}",
            "data_type": "medical_records"
        },
        "private_inputs": {
            "consent_signature": f"sig-{random.randint(1000, 9999)}",
            "timestamp": int(time.time()),
            "expiration": int(time.time() + 86400)
        }
    }
    
    response = http_post(ZK_EXECUTE_ENDPOINT, payload)
    if not response or response.status_code != 200:
        context.add_result("ZK Circuit Execution", False, 
                         f"Failed to execute ZK circuit. Status code: {response.status_code if response else 'No response'}")
        return
        
    execution_result = response.json()
    context.add_result("ZK Circuit Execution", True, 
                      f"Successfully executed ZK circuit", execution_result)
    
    # 3. Test proof verification
    if "proof" in execution_result:
        verify_payload = {
            "circuit_type": "patient-consent",
            "public_inputs": payload["public_inputs"],
            "proof": execution_result["proof"]
        }
        
        response = http_post(ZK_VERIFY_ENDPOINT, verify_payload)
        if response and response.status_code == 200:
            verification_result = response.json()
            context.add_result("ZK Proof Verification", True, 
                              f"Successfully verified ZK proof", verification_result)
        else:
            context.add_result("ZK Proof Verification", False, 
                              f"Failed to verify ZK proof. Status code: {response.status_code if response else 'No response'}")
    else:
        context.add_result("ZK Proof Verification", False, 
                          "No proof available to verify")

def validate_scaling_operations(context):
    """Validate scaling and load balancing operations"""
    logger.info("Validating scaling operations...")
    
    # 1. Check scaling status
    response = http_get(SCALING_STATUS_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("Scaling Status", False, 
                          f"Failed to get scaling status. Status code: {response.status_code if response else 'No response'}")
        return
        
    scaling_status = response.json()
    context.add_result("Scaling Status", True, 
                      f"Successfully retrieved scaling status", scaling_status)
    
    # 2. Check active nodes
    response = http_get(SCALING_NODES_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("Active Nodes Check", False, 
                          f"Failed to get active nodes. Status code: {response.status_code if response else 'No response'}")
        return
        
    nodes = response.json()
    context.add_result("Active Nodes Check", True, 
                      f"Found {len(nodes)} active nodes", nodes)
    
    # 3. Test node scaling (simulate high load)
    scale_payload = {
        "target_nodes": 3,
        "reason": "validation_test"
    }
    
    response = http_post(f"{BASE_URL}/scaling/scale", scale_payload)
    if response and response.status_code == 200:
        context.add_result("Node Scaling Test", True, 
                          f"Successfully requested node scaling", response.json())
        
        # Wait a moment for scaling to take effect
        time.sleep(5)
        
        # Check if nodes were actually scaled
        response = http_get(SCALING_NODES_ENDPOINT)
        if response and response.status_code == 200:
            updated_nodes = response.json()
            context.add_result("Node Scaling Verification", 
                              len(updated_nodes) >= scale_payload["target_nodes"],
                              f"Node count after scaling: {len(updated_nodes)}", updated_nodes)
    else:
        context.add_result("Node Scaling Test", False, 
                          f"Failed to request node scaling. Status code: {response.status_code if response else 'No response'}")

def validate_security_operations(context):
    """Validate security token generation and verification"""
    logger.info("Validating security operations...")
    
    # 1. Generate security token
    token_payload = {
        "subject": "validation-test",
        "scope": "api:read",
        "expiration": 3600  # 1 hour
    }
    
    response = http_post(SECURITY_TOKEN_ENDPOINT, token_payload)
    if not response or response.status_code != 200:
        context.add_result("Security Token Generation", False, 
                          f"Failed to generate token. Status code: {response.status_code if response else 'No response'}")
        return
        
    token_data = response.json()
    if "token" not in token_data:
        context.add_result("Security Token Generation", False, 
                          "No token in response")
        return
        
    context.add_result("Security Token Generation", True, 
                      "Successfully generated security token", token_data)
    
    # Store token for subsequent tests
    context.tokens["api_token"] = token_data["token"]
    
    # 2. Verify token
    headers = {"Authorization": f"Bearer {context.tokens['api_token']}"}
    response = http_get(f"{SECURITY_VERIFY_ENDPOINT}?token={context.tokens['api_token']}")
    
    if response and response.status_code == 200:
        verify_data = response.json()
        context.add_result("Security Token Verification", True, 
                          "Successfully verified security token", verify_data)
    else:
        context.add_result("Security Token Verification", False, 
                          f"Failed to verify token. Status code: {response.status_code if response else 'No response'}")

def validate_monitoring_operations(context):
    """Validate monitoring and health checking operations"""
    logger.info("Validating monitoring operations...")
    
    # 1. Check component health status
    response = http_get(MONITORING_HEALTH_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("Component Health Status", False, 
                          f"Failed to get component health. Status code: {response.status_code if response else 'No response'}")
    else:
        health_data = response.json()
        all_healthy = all(component["status"] == "healthy" for component in health_data["components"])
        
        context.add_result("Component Health Status", all_healthy, 
                          f"Component health check {'passed' if all_healthy else 'failed'}",
                          health_data)
    
    # 2. Check system metrics
    response = http_get(MONITORING_METRICS_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("System Metrics", False, 
                          f"Failed to get system metrics. Status code: {response.status_code if response else 'No response'}")
    else:
        metrics_data = response.json()
        context.add_result("System Metrics", True, 
                          "Successfully retrieved system metrics", metrics_data)
        
        # Store some key metrics
        if "system" in metrics_data:
            context.add_system_info("cpu_usage", metrics_data["system"].get("cpu_usage", "N/A"))
            context.add_system_info("memory_usage", metrics_data["system"].get("memory_usage", "N/A"))

def validate_interop_operations(context):
    """Validate interoperability with FHIR and EHR systems"""
    logger.info("Validating interoperability operations...")
    
    # 1. Check FHIR connectivity
    response = http_get(FHIR_STATUS_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("FHIR Connectivity", False, 
                          f"Failed to check FHIR status. Status code: {response.status_code if response else 'No response'}")
    else:
        fhir_status = response.json()
        context.add_result("FHIR Connectivity", fhir_status.get("status") == "connected", 
                          f"FHIR connection status: {fhir_status.get('status')}", fhir_status)
    
    # 2. Check EHR connectivity
    response = http_get(EHR_STATUS_ENDPOINT)
    if not response or response.status_code != 200:
        context.add_result("EHR Connectivity", False, 
                          f"Failed to check EHR status. Status code: {response.status_code if response else 'No response'}")
    else:
        ehr_status = response.json()
        context.add_result("EHR Connectivity", ehr_status.get("status") == "connected", 
                          f"EHR connection status: {ehr_status.get('status')}", ehr_status)
    
    # 3. Test FHIR patient resource creation
    patient_data = {
        "resourceType": "Patient",
        "name": [
            {
                "family": "Smith",
                "given": ["John"]
            }
        ],
        "gender": "male",
        "birthDate": "1970-01-01"
    }
    
    response = http_post(f"{BASE_URL}/interop/fhir/Patient", patient_data)
    if response and response.status_code in [200, 201]:
        context.add_result("FHIR Patient Creation", True, 
                          "Successfully created FHIR patient resource", response.json())
    else:
        context.add_result("FHIR Patient Creation", False, 
                          f"Failed to create FHIR patient. Status code: {response.status_code if response else 'No response'}")

def validate_policies(context):
    """Validate healthcare policy enforcement"""
    logger.info("Validating policy enforcement...")
    
    # 1. Basic policy validation - physician accessing patient records
    policy_payload = {
        "requester": {
            "id": "D1001",
            "role": "physician",
            "department": "Cardiology",
            "jurisdiction": "california"
        },
        "subject": {
            "id": "P2001",
            "record_type": "medical_history",
            "sensitivity": "high",
            "jurisdiction": "california"
        },
        "action": "read",
        "purpose": "treatment",
        "auth_method": "two_factor",
        "emergency": False
    }
    
    response = http_post(POLICY_VALIDATE_ENDPOINT, policy_payload)
    if not response or response.status_code != 200:
        context.add_result("Basic Policy Validation", False, 
                          f"Failed to validate policy. Status code: {response.status_code if response else 'No response'}")
    else:
        validation = response.json()
        context.add_result("Basic Policy Validation", validation.get("allowed", False), 
                          f"Policy validation: {validation.get('reason', 'Unknown')}", validation)
    
    # 2. Cross-jurisdiction validation
    cross_juris_payload = {
        "requester": {
            "id": "D1002",
            "role": "physician",
            "department": "Oncology",
            "jurisdiction": "california"
        },
        "subject": {
            "id": "P2002",
            "record_type": "medical_history",
            "sensitivity": "high",
            "jurisdiction": "new_york"
        },
        "action": "read",
        "purpose": "treatment",
        "auth_method": "two_factor",
        "emergency": False
    }
    
    response = http_post(POLICY_VALIDATE_ENDPOINT, cross_juris_payload)
    if response and response.status_code == 200:
        validation = response.json()
        context.add_result("Cross-Jurisdiction Policy", validation.get("allowed", False), 
                          f"Cross-jurisdiction validation: {validation.get('reason', 'Unknown')}", validation)
    else:
        context.add_result("Cross-Jurisdiction Policy", False, 
                          f"Failed to validate cross-jurisdiction policy. Status code: {response.status_code if response else 'No response'}")
    
    # 3. Role-based policy validation
    for role in ["physician", "nurse", "researcher", "insurance_agent"]:
        role_payload = {
            "requester": {
                "id": f"D{random.randint(1000, 9999)}",
                "role": role,
                "department": "General",
                "jurisdiction": "california"
            },
            "subject": {
                "id": "P2003",
                "record_type": "medical_history",
                "sensitivity": "medium",
                "jurisdiction": "california"
            },
            "action": "read",
            "purpose": "treatment"
        }
        
        response = http_post(POLICY_ROLE_ENDPOINT, role_payload)
        if response and response.status_code == 200:
            validation = response.json()
            context.add_result(f"Role-Based Policy: {role}", validation.get("allowed", False), 
                              f"Role '{role}' validation: {validation.get('reason', 'Unknown')}", validation)
        else:
            context.add_result(f"Role-Based Policy: {role}", False, 
                              f"Failed to validate {role} policy. Status code: {response.status_code if response else 'No response'}")
    
    # 4. Emergency access policy
    emergency_payload = {
        "requester": {
            "id": "D1004",
            "role": "nurse",
            "department": "Emergency",
            "jurisdiction": "texas"
        },
        "subject": {
            "id": "P2004",
            "record_type": "medical_history",
            "sensitivity": "high",
            "jurisdiction": "florida"
        },
        "action": "read",
        "purpose": "emergency",
        "auth_method": "password",
        "emergency": True
    }
    
    response = http_post(POLICY_VALIDATE_ENDPOINT, emergency_payload)
    if response and response.status_code == 200:
        validation = response.json()
        context.add_result("Emergency Access Policy", validation.get("allowed", False), 
                          f"Emergency access validation: {validation.get('reason', 'Unknown')}", validation)
    else:
        context.add_result("Emergency Access Policy", False, 
                          f"Failed to validate emergency policy. Status code: {response.status_code if response else 'No response'}")

def validate_load_testing(context):
    """Perform load testing to validate infrastructure scalability"""
    logger.info("Performing load testing...")
    
    NUM_REQUESTS = 100
    MAX_WORKERS = 10
    
    def make_zk_request():
        """Execute a ZK circuit as a load test request"""
        payload = {
            "circuit_type": "patient-consent",
            "public_inputs": {
                "patient_id": f"P{random.randint(1000, 9999)}",
                "provider_id": f"D{random.randint(1000, 9999)}",
                "data_type": "medical_records"
            },
            "private_inputs": {
                "consent_signature": f"sig-{random.randint(1000, 9999)}",
                "timestamp": int(time.time()),
                "expiration": int(time.time() + 86400)
            }
        }
        
        start_time = time.time()
        response = http_post(ZK_EXECUTE_ENDPOINT, payload)
        end_time = time.time()
        
        if response and response.status_code == 200:
            return {
                "success": True,
                "response_time": (end_time - start_time) * 1000  # Convert to ms
            }
        else:
            return {
                "success": False,
                "response_time": (end_time - start_time) * 1000,
                "status_code": response.status_code if response else None
            }
    
    # Execute requests in parallel
    start_time = time.time()
    results = []
    
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        future_to_request = {executor.submit(make_zk_request): i for i in range(NUM_REQUESTS)}
        for future in future_to_request:
            results.append(future.result())
    
    end_time = time.time()
    
    # Calculate statistics
    successful_requests = [r for r in results if r["success"]]
    failed_requests = [r for r in results if not r["success"]]
    
    if successful_requests:
        avg_response_time = sum(r["response_time"] for r in successful_requests) / len(successful_requests)
        min_response_time = min(r["response_time"] for r in successful_requests)
        max_response_time = max(r["response_time"] for r in successful_requests)
    else:
        avg_response_time = min_response_time = max_response_time = 0
    
    total_time = end_time - start_time
    throughput = NUM_REQUESTS / total_time if total_time > 0 else 0
    
    load_test_results = {
        "total_requests": NUM_REQUESTS,
        "successful_requests": len(successful_requests),
        "failed_requests": len(failed_requests),
        "total_time_seconds": total_time,
        "avg_response_time_ms": avg_response_time,
        "min_response_time_ms": min_response_time,
        "max_response_time_ms": max_response_time,
        "throughput_rps": throughput
    }
    
    context.add_result("Load Testing", len(failed_requests) == 0, 
                      f"Load test completed with {len(successful_requests)}/{NUM_REQUESTS} successful requests", 
                      load_test_results)

def main():
    """Main validation function"""
    parser = argparse.ArgumentParser(description="ZK-Proof Healthcare Infrastructure Validation Tool")
    parser.add_argument("--skip-load-test", action="store_true", help="Skip load testing")
    args = parser.parse_args()
    
    context = ValidationContext()
    logger.info("Starting infrastructure validation...")
    
    # Run validation tests
    validate_infrastructure_health(context)
    validate_zk_circuit_operations(context)
    validate_scaling_operations(context)
    validate_security_operations(context)
    validate_monitoring_operations(context)
    validate_interop_operations(context)
    validate_policies(context)
    
    # Run load testing if not skipped
    if not args.skip_load_test:
        validate_load_testing(context)
    
    # Generate and save summary report
    summary = context.get_summary()
    logger.info(f"Validation complete. Success rate: {summary['summary']['success_rate']}")
    logger.info(f"Total tests: {summary['summary']['total_tests']}, "
               f"Passed: {summary['summary']['passed_tests']}, "
               f"Failed: {summary['summary']['failed_tests']}")
    
    # Save detailed report to file
    with open('validation_report.json', 'w') as f:
        json.dump(summary, f, indent=2)
    logger.info("Detailed report saved to validation_report.json")
    
    # Return success status for integration with CI/CD pipelines
    return summary['summary']['failed_tests'] == 0

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
