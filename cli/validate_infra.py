#!/usr/bin/env python3
"""
Validation script for the ZK Health Infrastructure
This script runs a non-interactive version of the demo to verify all components
"""

import sys
import traceback
from rich.console import Console
from rich.panel import Panel
from demo import initialize_demo_data, step_1_register_identities, step_2_create_oracle_agreement
from demo import step_3_establish_consent, step_4_document_upload, step_5_treatment_vector, step_6_api_gateway

console = Console()

def validate_infrastructure():
    """
    Run through all components in sequence and verify they work properly
    """
    console.print(Panel.fit(
        "[bold]ZK Health Infrastructure Validation[/bold]\n\n"
        "This tool will verify all components of the ZK-based healthcare infrastructure",
        title="Infrastructure Validation", border_style="blue"
    ))
    
    try:
        # Initialize test data
        console.print("[bold]1. Initializing test data...[/bold]")
        demo_data = initialize_demo_data()
        console.print("[green]✓ Test data initialized successfully[/green]")
        
        # Validate identity management
        console.print("\n[bold]2. Validating ZK Identity Management...[/bold]")
        step_1_register_identities(console, demo_data)
        console.print("[green]✓ ZK Identity Management validated[/green]")
        
        # Validate oracle agreements
        console.print("\n[bold]3. Validating Oracle Chain Validator...[/bold]")
        step_2_create_oracle_agreement(console, demo_data)
        console.print("[green]✓ Oracle Chain Validator validated[/green]")
        
        # Validate consent management
        console.print("\n[bold]4. Validating Consent Management...[/bold]")
        step_3_establish_consent(console, demo_data)
        console.print("[green]✓ Consent Management validated[/green]")
        
        # Validate document archive
        console.print("\n[bold]5. Validating Cassandra Document Archive...[/bold]")
        step_4_document_upload(console, demo_data)
        console.print("[green]✓ Cassandra Document Archive validated[/green]")
        
        # Validate treatment vectors
        console.print("\n[bold]6. Validating YAG AI & Treatment Vectors...[/bold]")
        step_5_treatment_vector(console, demo_data)
        console.print("[green]✓ YAG AI & Treatment Vectors validated[/green]")
        
        # Validate API gateway
        console.print("\n[bold]7. Validating ZK API Gateway...[/bold]")
        step_6_api_gateway(console, demo_data)
        console.print("[green]✓ ZK API Gateway validated[/green]")
        
        # Final validation
        console.print("\n[bold green]✓ All components validated successfully![/bold green]")
        console.print("\nGenerated entities during validation:")
        console.print(f"  Agreement ID: {demo_data['generated']['agreement_id']}")
        console.print(f"  Consent ID: {demo_data['generated']['consent_id']}")
        console.print(f"  Document ID: {demo_data['generated']['document_id']}")
        console.print(f"  Vector ID: {demo_data['generated']['vector_id']}")
        console.print(f"  Token ID: {demo_data['generated']['token_id']}")
        
        return True
    except Exception as e:
        console.print(f"\n[bold red]✗ Validation failed![/bold red]")
        console.print(f"Error: {str(e)}")
        console.print("Stack trace:")
        traceback.print_exc()
        return False

if __name__ == "__main__":
    success = validate_infrastructure()
    sys.exit(0 if success else 1)
