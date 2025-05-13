#!/usr/bin/env python3
"""
ZK Health CLI - Test and demo tool for ZK-based Decentralized Healthcare Infrastructure
"""

import os
import sys
import json
import click
import dotenv
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.markdown import Markdown

# Import CLI modules
from identity_commands import identity_group
from consent_commands import consent_group
from oracle_commands import oracle_group 
from document_commands import document_group
from treatment_commands import treatment_group
from gateway_commands import gateway_group

# Load environment variables from .env file if it exists
dotenv.load_dotenv()

# Initialize rich console for pretty output
console = Console()

@click.group()
@click.version_option(version="1.0.0")
def cli():
    """
    ZK Health CLI - Test and demo tool for the ZK-based Decentralized Healthcare Infrastructure.
    
    This CLI allows you to interact with all components of the telemedicine system:
    - ZK Identity Management
    - Consent Management
    - Oracle Agreements
    - Document Storage
    - Treatment Vectors
    - API Gateway
    """
    pass

@cli.command()
def status():
    """Check the status of the ZK Health services"""
    from utils import check_service_status
    
    console.print(Panel.fit("ZK Health Infrastructure Status", style="bold magenta"))
    
    services = [
        {"name": "API Server", "url": os.getenv("API_URL", "http://localhost:8080/health")},
        {"name": "MongoDB", "url": os.getenv("MONGO_URL", "http://localhost:27017")},
        {"name": "Cassandra", "url": os.getenv("CASSANDRA_URL", "http://localhost:9042")}
    ]
    
    table = Table(show_header=True, header_style="bold blue")
    table.add_column("Service")
    table.add_column("Status")
    table.add_column("Details")
    
    for service in services:
        status, details = check_service_status(service["url"])
        status_color = "green" if status else "red"
        table.add_row(
            service["name"],
            f"[{status_color}]{'ONLINE' if status else 'OFFLINE'}[/{status_color}]",
            details
        )
    
    console.print(table)

@cli.command()
def showcase():
    """Showcase the capabilities of the ZK Health infrastructure"""
    showcase_text = """
    # 🚀 ZK-Based Decentralized Healthcare Infrastructure
    
    ## Core Components:
    
    - **ZK-Proof Identity Management** - Secure identity verification without revealing PII
    - **Multi-Party Consent Framework** - Patient-centric data access control
    - **Oracle Agreement Engine** - Dynamic regulatory compliance
    - **Tamper-Proof Document Archive** - Securely store and verify medical records
    - **Treatment Vector Misalignment** - AI-assisted treatment verification
    - **API Gateway with ZK Authentication** - Secure access to the infrastructure
    
    ## Unique Capabilities:
    
    - Cross-border compliance with local regulations
    - Full audit trails with cryptographic proof
    - Patient-controlled health data
    - AI-augmented diagnostic and treatment paths
    - Regulatory updates without code changes
    """
    
    console.print(Markdown(showcase_text))
    
    console.print("\n[bold]Use the following commands to explore the infrastructure:[/bold]")
    console.print("  • zk_health_cli.py identity --help")
    console.print("  • zk_health_cli.py consent --help")
    console.print("  • zk_health_cli.py oracle --help")
    console.print("  • zk_health_cli.py document --help")
    console.print("  • zk_health_cli.py treatment --help")
    console.print("  • zk_health_cli.py gateway --help")
    
    console.print("\n[bold]Example workflow:[/bold]")
    console.print("  1. Create identities for doctor and patient")
    console.print("  2. Create consent agreement between them")
    console.print("  3. Register oracle agreement for medical consultation")
    console.print("  4. Upload and verify medical documents")
    console.print("  5. Start and track treatment vector")
    console.print("  6. Generate and validate API tokens")

@cli.command()
def demo():
    """Run an automated demo of key workflows"""
    from demo import run_demo
    run_demo(console)

# Register command groups
cli.add_command(identity_group)
cli.add_command(consent_group)
cli.add_command(oracle_group)
cli.add_command(document_group)
cli.add_command(treatment_group)
cli.add_command(gateway_group)

if __name__ == "__main__":
    cli()
