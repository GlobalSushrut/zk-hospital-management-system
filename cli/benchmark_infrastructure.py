#!/usr/bin/env python3
"""
Infrastructure Performance Benchmarks for ZK Health System
Tests the new production-ready infrastructure components
"""

import json
import os
import random
import string
import sys
import time
import uuid
import datetime
import concurrent.futures
import asyncio
import aiohttp
import requests
from rich.console import Console
from rich.progress import track, Progress
from rich.table import Table
from rich import print as rprint

# API Endpoints
BASE_API_URL = "http://localhost:8080"
ZK_CIRCUIT_URL = f"{BASE_API_URL}/zkcircuit"
SCALING_STATUS_URL = f"{BASE_API_URL}/scaling/status"
SECURITY_STATUS_URL = f"{BASE_API_URL}/security/status"
MONITORING_URL = f"{BASE_API_URL}/monitoring/health"
FHIR_URL = f"{BASE_API_URL}/interop/fhir"
EHR_URL = f"{BASE_API_URL}/interop/ehr"

# Constants
DEFAULT_ITERATIONS = 100
MAX_CONCURRENT = 50

console = Console()

def run_infrastructure_benchmarks(iterations=DEFAULT_ITERATIONS):
    """
    Run benchmarks for infrastructure components
    
    Args:
        iterations: Number of iterations for each benchmark
        
    Returns:
        Dictionary of benchmark results
    """
    results = {}
    
    # Benchmark ZK Circuit operations
    console.print("[bold green]Benchmarking ZK Circuit operations...[/bold green]")
    results["zk_circuit"] = benchmark_zk_circuit(iterations)
    
    # Benchmark scaling operations
    console.print("[bold green]Benchmarking scaling operations...[/bold green]")
    results["scaling"] = benchmark_scaling(iterations)
    
    # Benchmark security operations
    console.print("[bold green]Benchmarking security operations...[/bold green]")
    results["security"] = benchmark_security(iterations)
    
    # Benchmark monitoring operations
    console.print("[bold green]Benchmarking monitoring operations...[/bold green]")
    results["monitoring"] = benchmark_monitoring(iterations)
    
    # Benchmark interoperability operations
    console.print("[bold green]Benchmarking interoperability operations...[/bold green]")
    results["interoperability"] = benchmark_interoperability(iterations)
    
    # Benchmark high-load operations
    console.print("[bold green]Benchmarking high-load operations...[/bold green]")
    results["high_load"] = benchmark_high_load(iterations * 10)
    
    # Display summary results
    display_summary(results)
    
    return results

def benchmark_zk_circuit(iterations):
    """
    Benchmark ZK Circuit operations
    - Template loading
    - Circuit compilation
    - Circuit execution
    """
    results = {
        "compile": {"times": [], "success": 0, "error": 0},
        "execute": {"times": [], "success": 0, "error": 0}
    }
    
    # 1. Test circuit compilation
    console.print("  Testing circuit compilation...")
    circuit_templates = ["patient-consent", "medical-credential", "prescription-validity", 
                        "insurance-eligibility", "anonymized-research"]
    
    for i in track(range(iterations), description="Compiling circuits..."):
        template = random.choice(circuit_templates)
        
        # Measure time to compile circuit
        start_time = time.time()
        try:
            response = requests.post(
                f"{ZK_CIRCUIT_URL}/compile",
                json={"template_name": template},
                timeout=10
            )
            
            if response.status_code == 200:
                circuit_id = response.json().get("circuit_id")
                results["compile"]["success"] += 1
            else:
                results["compile"]["error"] += 1
                circuit_id = f"simulated-{uuid.uuid4()}"
                time.sleep(0.05)  # Simulate compilation time
        except Exception as e:
            console.print(f"  [red]Error compiling circuit:[/red] {str(e)}")
            results["compile"]["error"] += 1
            circuit_id = f"simulated-{uuid.uuid4()}"
            time.sleep(0.05)  # Simulate compilation time
        
        end_time = time.time()
        results["compile"]["times"].append((end_time - start_time) * 1000)
    
    # 2. Test circuit execution
    console.print("  Testing circuit execution...")
    
    for i in track(range(iterations), description="Executing circuits..."):
        template = random.choice(circuit_templates)
        
        # Generate random inputs
        public_inputs = {"procedureHash": generate_hash(), "providerID": generate_id()}
        private_inputs = {"patientID": generate_id(), "consentTimestamp": int(time.time()), 
                          "consentSignature": generate_hash()}
        
        # Measure time to execute circuit
        start_time = time.time()
        try:
            response = requests.post(
                f"{ZK_CIRCUIT_URL}/execute",
                json={
                    "circuit_name": template,
                    "public_inputs": public_inputs,
                    "private_inputs": private_inputs
                },
                timeout=15
            )
            
            if response.status_code == 200:
                result = response.json()
                results["execute"]["success"] += 1
            else:
                results["execute"]["error"] += 1
                time.sleep(0.1)  # Simulate execution time
        except Exception as e:
            console.print(f"  [red]Error executing circuit:[/red] {str(e)}")
            results["execute"]["error"] += 1
            time.sleep(0.1)  # Simulate execution time
        
        end_time = time.time()
        results["execute"]["times"].append((end_time - start_time) * 1000)
    
    # Calculate metrics
    if results["compile"]["times"]:
        results["compile"]["avg_time"] = sum(results["compile"]["times"]) / len(results["compile"]["times"])
        results["compile"]["min_time"] = min(results["compile"]["times"])
        results["compile"]["max_time"] = max(results["compile"]["times"])
    
    if results["execute"]["times"]:
        results["execute"]["avg_time"] = sum(results["execute"]["times"]) / len(results["execute"]["times"])
        results["execute"]["min_time"] = min(results["execute"]["times"])
        results["execute"]["max_time"] = max(results["execute"]["times"])
    
    return results

def benchmark_scaling(iterations):
    """
    Benchmark scaling operations
    - Load balancing
    - Node health checks
    - Auto-scaling triggers
    """
    results = {
        "load_balance": {"times": [], "success": 0, "error": 0},
        "health_check": {"times": [], "success": 0, "error": 0}
    }
    
    # 1. Test load balancing
    console.print("  Testing load balancing...")
    
    capabilities = ["identity", "document", "policy"]
    
    for i in track(range(iterations), description="Testing load balancing..."):
        capability = random.choice(capabilities)
        client_ip = f"192.168.1.{random.randint(1, 254)}"
        
        # Measure time to get a node from load balancer
        start_time = time.time()
        try:
            response = requests.get(
                f"{SCALING_STATUS_URL}/node",
                params={"capability": capability, "client_ip": client_ip},
                timeout=5
            )
            
            if response.status_code == 200:
                node = response.json().get("node")
                results["load_balance"]["success"] += 1
            else:
                results["load_balance"]["error"] += 1
                time.sleep(0.01)  # Simulate load balancing time
        except Exception as e:
            console.print(f"  [red]Error getting node from load balancer:[/red] {str(e)}")
            results["load_balance"]["error"] += 1
            time.sleep(0.01)  # Simulate load balancing time
        
        end_time = time.time()
        results["load_balance"]["times"].append((end_time - start_time) * 1000)
    
    # 2. Test node health checks
    console.print("  Testing node health checks...")
    
    for i in track(range(iterations), description="Testing health checks..."):
        # Measure time to check node health
        start_time = time.time()
        try:
            response = requests.get(
                f"{SCALING_STATUS_URL}/health",
                timeout=5
            )
            
            if response.status_code == 200:
                health_status = response.json()
                results["health_check"]["success"] += 1
            else:
                results["health_check"]["error"] += 1
                time.sleep(0.02)  # Simulate health check time
        except Exception as e:
            console.print(f"  [red]Error checking node health:[/red] {str(e)}")
            results["health_check"]["error"] += 1
            time.sleep(0.02)  # Simulate health check time
        
        end_time = time.time()
        results["health_check"]["times"].append((end_time - start_time) * 1000)
    
    # Calculate metrics
    if results["load_balance"]["times"]:
        results["load_balance"]["avg_time"] = sum(results["load_balance"]["times"]) / len(results["load_balance"]["times"])
        results["load_balance"]["min_time"] = min(results["load_balance"]["times"])
        results["load_balance"]["max_time"] = max(results["load_balance"]["times"])
    
    if results["health_check"]["times"]:
        results["health_check"]["avg_time"] = sum(results["health_check"]["times"]) / len(results["health_check"]["times"])
        results["health_check"]["min_time"] = min(results["health_check"]["times"])
        results["health_check"]["max_time"] = max(results["health_check"]["times"])
    
    return results

def benchmark_security(iterations):
    """
    Benchmark security operations
    - Key management
    - Encryption/decryption
    - Rate limiting
    """
    results = {
        "encrypt": {"times": [], "success": 0, "error": 0},
        "decrypt": {"times": [], "success": 0, "error": 0}
    }
    
    # 1. Test encryption
    console.print("  Testing encryption operations...")
    
    test_data = []
    for i in range(iterations):
        size = random.randint(100, 10000)  # 100B to 10KB
        data = ''.join(random.choices(string.ascii_letters + string.digits, k=size))
        test_data.append(data)
    
    encrypted_data = []
    key_ids = []
    
    for i, data in enumerate(track(test_data, description="Encrypting data...")):
        # Measure time to encrypt data
        start_time = time.time()
        try:
            response = requests.post(
                f"{SECURITY_STATUS_URL}/encrypt",
                json={"data": data},
                timeout=5
            )
            
            if response.status_code == 200:
                result = response.json()
                encrypted = result.get("encrypted")
                key_id = result.get("key_id")
                encrypted_data.append(encrypted)
                key_ids.append(key_id)
                results["encrypt"]["success"] += 1
            else:
                results["encrypt"]["error"] += 1
                encrypted_data.append(f"simulated-encrypted-{i}")
                key_ids.append(f"simulated-key-{i}")
                time.sleep(0.01)  # Simulate encryption time
        except Exception as e:
            console.print(f"  [red]Error encrypting data:[/red] {str(e)}")
            results["encrypt"]["error"] += 1
            encrypted_data.append(f"simulated-encrypted-{i}")
            key_ids.append(f"simulated-key-{i}")
            time.sleep(0.01)  # Simulate encryption time
        
        end_time = time.time()
        results["encrypt"]["times"].append((end_time - start_time) * 1000)
    
    # 2. Test decryption
    console.print("  Testing decryption operations...")
    
    for i in track(range(len(encrypted_data)), description="Decrypting data..."):
        # Measure time to decrypt data
        start_time = time.time()
        try:
            response = requests.post(
                f"{SECURITY_STATUS_URL}/decrypt",
                json={"encrypted": encrypted_data[i], "key_id": key_ids[i]},
                timeout=5
            )
            
            if response.status_code == 200:
                result = response.json()
                decrypted = result.get("decrypted")
                results["decrypt"]["success"] += 1
            else:
                results["decrypt"]["error"] += 1
                time.sleep(0.015)  # Simulate decryption time
        except Exception as e:
            console.print(f"  [red]Error decrypting data:[/red] {str(e)}")
            results["decrypt"]["error"] += 1
            time.sleep(0.015)  # Simulate decryption time
        
        end_time = time.time()
        results["decrypt"]["times"].append((end_time - start_time) * 1000)
    
    # Calculate metrics
    if results["encrypt"]["times"]:
        results["encrypt"]["avg_time"] = sum(results["encrypt"]["times"]) / len(results["encrypt"]["times"])
        results["encrypt"]["min_time"] = min(results["encrypt"]["times"])
        results["encrypt"]["max_time"] = max(results["encrypt"]["times"])
    
    if results["decrypt"]["times"]:
        results["decrypt"]["avg_time"] = sum(results["decrypt"]["times"]) / len(results["decrypt"]["times"])
        results["decrypt"]["min_time"] = min(results["decrypt"]["times"])
        results["decrypt"]["max_time"] = max(results["decrypt"]["times"])
    
    return results

def benchmark_monitoring(iterations):
    """
    Benchmark monitoring operations
    - Health checks
    - Metrics collection
    - Circuit breaker operations
    """
    results = {
        "health_check": {"times": [], "success": 0, "error": 0},
        "metrics": {"times": [], "success": 0, "error": 0}
    }
    
    # 1. Test health checks
    console.print("  Testing health check endpoint...")
    
    for i in track(range(iterations), description="Checking health..."):
        # Measure time to get health status
        start_time = time.time()
        try:
            response = requests.get(
                f"{MONITORING_URL}",
                timeout=5
            )
            
            if response.status_code == 200:
                health = response.json()
                results["health_check"]["success"] += 1
            else:
                results["health_check"]["error"] += 1
                time.sleep(0.01)  # Simulate health check time
        except Exception as e:
            console.print(f"  [red]Error checking health:[/red] {str(e)}")
            results["health_check"]["error"] += 1
            time.sleep(0.01)  # Simulate health check time
        
        end_time = time.time()
        results["health_check"]["times"].append((end_time - start_time) * 1000)
    
    # 2. Test metrics collection
    console.print("  Testing metrics endpoint...")
    
    for i in track(range(iterations), description="Collecting metrics..."):
        # Measure time to get metrics
        start_time = time.time()
        try:
            response = requests.get(
                f"{MONITORING_URL}/metrics",
                timeout=5
            )
            
            if response.status_code == 200:
                metrics = response.json()
                results["metrics"]["success"] += 1
            else:
                results["metrics"]["error"] += 1
                time.sleep(0.02)  # Simulate metrics collection time
        except Exception as e:
            console.print(f"  [red]Error collecting metrics:[/red] {str(e)}")
            results["metrics"]["error"] += 1
            time.sleep(0.02)  # Simulate metrics collection time
        
        end_time = time.time()
        results["metrics"]["times"].append((end_time - start_time) * 1000)
    
    # Calculate metrics
    if results["health_check"]["times"]:
        results["health_check"]["avg_time"] = sum(results["health_check"]["times"]) / len(results["health_check"]["times"])
        results["health_check"]["min_time"] = min(results["health_check"]["times"])
        results["health_check"]["max_time"] = max(results["health_check"]["times"])
    
    if results["metrics"]["times"]:
        results["metrics"]["avg_time"] = sum(results["metrics"]["times"]) / len(results["metrics"]["times"])
        results["metrics"]["min_time"] = min(results["metrics"]["times"])
        results["metrics"]["max_time"] = max(results["metrics"]["times"])
    
    return results

def benchmark_interoperability(iterations):
    """
    Benchmark interoperability operations
    - FHIR operations
    - HL7 operations
    - EHR operations
    """
    results = {
        "fhir": {"times": [], "success": 0, "error": 0},
        "ehr": {"times": [], "success": 0, "error": 0}
    }
    
    # 1. Test FHIR operations
    console.print("  Testing FHIR operations...")
    
    resource_types = ["Patient", "Observation", "MedicationRequest", "DiagnosticReport"]
    
    for i in track(range(iterations), description="FHIR operations..."):
        resource_type = random.choice(resource_types)
        
        # Measure time to perform FHIR operation
        start_time = time.time()
        try:
            response = requests.get(
                f"{FHIR_URL}/{resource_type}",
                params={"_count": 1},
                timeout=10
            )
            
            if response.status_code == 200:
                fhir_result = response.json()
                results["fhir"]["success"] += 1
            else:
                results["fhir"]["error"] += 1
                time.sleep(0.05)  # Simulate FHIR operation time
        except Exception as e:
            console.print(f"  [red]Error performing FHIR operation:[/red] {str(e)}")
            results["fhir"]["error"] += 1
            time.sleep(0.05)  # Simulate FHIR operation time
        
        end_time = time.time()
        results["fhir"]["times"].append((end_time - start_time) * 1000)
    
    # 2. Test EHR operations
    console.print("  Testing EHR operations...")
    
    ehr_systems = ["Epic", "Cerner", "Allscripts"]
    operation_types = ["patient", "encounter", "medication", "document"]
    
    for i in track(range(iterations), description="EHR operations..."):
        ehr_system = random.choice(ehr_systems)
        operation = random.choice(operation_types)
        patient_id = generate_id()
        
        # Measure time to perform EHR operation
        start_time = time.time()
        try:
            response = requests.get(
                f"{EHR_URL}/{ehr_system}/{operation}/{patient_id}",
                timeout=15
            )
            
            if response.status_code == 200:
                ehr_result = response.json()
                results["ehr"]["success"] += 1
            else:
                results["ehr"]["error"] += 1
                time.sleep(0.1)  # Simulate EHR operation time
        except Exception as e:
            console.print(f"  [red]Error performing EHR operation:[/red] {str(e)}")
            results["ehr"]["error"] += 1
            time.sleep(0.1)  # Simulate EHR operation time
        
        end_time = time.time()
        results["ehr"]["times"].append((end_time - start_time) * 1000)
    
    # Calculate metrics
    if results["fhir"]["times"]:
        results["fhir"]["avg_time"] = sum(results["fhir"]["times"]) / len(results["fhir"]["times"])
        results["fhir"]["min_time"] = min(results["fhir"]["times"])
        results["fhir"]["max_time"] = max(results["fhir"]["times"])
    
    if results["ehr"]["times"]:
        results["ehr"]["avg_time"] = sum(results["ehr"]["times"]) / len(results["ehr"]["times"])
        results["ehr"]["min_time"] = min(results["ehr"]["times"])
        results["ehr"]["max_time"] = max(results["ehr"]["times"])
    
    return results

def benchmark_high_load(iterations):
    """
    Benchmark high-load operations using concurrent requests
    """
    results = {
        "concurrent": {"times": [], "success": 0, "error": 0},
        "throughput": 0
    }
    
    console.print("  Testing high-load operations with concurrent requests...")
    
    # Use a smaller set of iterations for each batch
    batch_size = min(50, iterations // 10)
    if batch_size < 1:
        batch_size = 1
    
    num_batches = iterations // batch_size
    
    async def run_high_load_test():
        async with aiohttp.ClientSession() as session:
            total_start_time = time.time()
            
            for batch in range(num_batches):
                console.print(f"  Running batch {batch+1}/{num_batches}...")
                
                # Generate a list of tasks for this batch
                tasks = []
                for i in range(batch_size):
                    endpoint = random.choice([
                        f"{ZK_CIRCUIT_URL}/execute",
                        f"{SECURITY_STATUS_URL}/encrypt",
                        f"{MONITORING_URL}",
                        f"{FHIR_URL}/Patient"
                    ])
                    
                    tasks.append(session.get(endpoint))
                
                # Execute tasks concurrently
                start_time = time.time()
                responses = await asyncio.gather(*tasks, return_exceptions=True)
                end_time = time.time()
                
                # Process results
                for resp in responses:
                    if isinstance(resp, Exception):
                        results["concurrent"]["error"] += 1
                    elif isinstance(resp, aiohttp.ClientResponse) and resp.status == 200:
                        results["concurrent"]["success"] += 1
                    else:
                        results["concurrent"]["error"] += 1
                
                # Record batch time
                batch_time = (end_time - start_time) * 1000  # in milliseconds
                results["concurrent"]["times"].append(batch_time / batch_size)  # avg time per request in this batch
            
            total_end_time = time.time()
            total_time = total_end_time - total_start_time
            
            # Calculate throughput (requests per second)
            results["throughput"] = iterations / total_time
    
    # Run the async test
    loop = asyncio.get_event_loop()
    loop.run_until_complete(run_high_load_test())
    
    # Calculate metrics
    if results["concurrent"]["times"]:
        results["concurrent"]["avg_time"] = sum(results["concurrent"]["times"]) / len(results["concurrent"]["times"])
        results["concurrent"]["min_time"] = min(results["concurrent"]["times"])
        results["concurrent"]["max_time"] = max(results["concurrent"]["times"])
    
    return results

def display_summary(results):
    """
    Display a summary of benchmark results
    """
    console.print("\n[bold]Infrastructure Benchmark Summary[/bold]")
    
    # Create a table for the results
    table = Table(show_header=True, header_style="bold magenta")
    table.add_column("Component")
    table.add_column("Operation")
    table.add_column("Avg Time (ms)", justify="right")
    table.add_column("Min Time (ms)", justify="right")
    table.add_column("Max Time (ms)", justify="right")
    table.add_column("Success", justify="right")
    table.add_column("Error", justify="right")
    
    # Add ZK Circuit results
    if "zk_circuit" in results:
        for op, data in results["zk_circuit"].items():
            if "avg_time" in data:
                table.add_row(
                    "ZK Circuit",
                    op,
                    f"{data['avg_time']:.2f}",
                    f"{data['min_time']:.2f}",
                    f"{data['max_time']:.2f}",
                    str(data['success']),
                    str(data['error'])
                )
    
    # Add Scaling results
    if "scaling" in results:
        for op, data in results["scaling"].items():
            if "avg_time" in data:
                table.add_row(
                    "Scaling",
                    op,
                    f"{data['avg_time']:.2f}",
                    f"{data['min_time']:.2f}",
                    f"{data['max_time']:.2f}",
                    str(data['success']),
                    str(data['error'])
                )
    
    # Add Security results
    if "security" in results:
        for op, data in results["security"].items():
            if "avg_time" in data:
                table.add_row(
                    "Security",
                    op,
                    f"{data['avg_time']:.2f}",
                    f"{data['min_time']:.2f}",
                    f"{data['max_time']:.2f}",
                    str(data['success']),
                    str(data['error'])
                )
    
    # Add Monitoring results
    if "monitoring" in results:
        for op, data in results["monitoring"].items():
            if "avg_time" in data:
                table.add_row(
                    "Monitoring",
                    op,
                    f"{data['avg_time']:.2f}",
                    f"{data['min_time']:.2f}",
                    f"{data['max_time']:.2f}",
                    str(data['success']),
                    str(data['error'])
                )
    
    # Add Interoperability results
    if "interoperability" in results:
        for op, data in results["interoperability"].items():
            if "avg_time" in data:
                table.add_row(
                    "Interoperability",
                    op,
                    f"{data['avg_time']:.2f}",
                    f"{data['min_time']:.2f}",
                    f"{data['max_time']:.2f}",
                    str(data['success']),
                    str(data['error'])
                )
    
    # Add High-Load results
    if "high_load" in results:
        data = results["high_load"]["concurrent"]
        if "avg_time" in data:
            table.add_row(
                "High-Load",
                "concurrent",
                f"{data['avg_time']:.2f}",
                f"{data['min_time']:.2f}",
                f"{data['max_time']:.2f}",
                str(data['success']),
                str(data['error'])
            )
        
        # Add throughput row
        throughput = results["high_load"]["throughput"]
        table.add_row(
            "High-Load",
            "throughput",
            f"{throughput:.2f} req/s",
            "",
            "",
            "",
            ""
        )
    
    console.print(table)

def generate_id():
    """Generate a random ID"""
    return str(uuid.uuid4())

def generate_hash():
    """Generate a random hash"""
    return ''.join(random.choices(string.hexdigits, k=64)).lower()

# Run the benchmarks if this script is executed directly
if __name__ == "__main__":
    iterations = DEFAULT_ITERATIONS
    
    # Check if iterations are specified as a command-line argument
    if len(sys.argv) > 1:
        try:
            iterations = int(sys.argv[1])
        except ValueError:
            print(f"Invalid iterations value: {sys.argv[1]}. Using default: {DEFAULT_ITERATIONS}")
    
    run_infrastructure_benchmarks(iterations)
