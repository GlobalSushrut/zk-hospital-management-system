"""
Identity commands for ZK Health CLI
"""

import os
import json
import click
from rich.console import Console
from rich.table import Table

from utils import make_api_request, handle_api_error

console = Console()

@click.group(name="identity")
def identity_group():
    """Manage ZK Identities"""
    pass

@identity_group.command(name="create")
@click.option('--party-id', required=True, help='Unique identifier for the party')
@click.option('--claim', required=True, help='Identity claim (e.g., doctor, patient, admin)')
@click.option('--name', required=True, help='Name of the identity holder')
@click.option('--metadata', help='Additional metadata as JSON string')
def create_identity(party_id, claim, name, metadata):
    """Register a new identity with ZK proof"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        meta_data = {}
        if metadata:
            meta_data = json.loads(metadata)
        
        meta_data['name'] = name
        
        payload = {
            'party_id': party_id,
            'claim': claim,
            'metadata': meta_data
        }
        
        response = make_api_request(f"{api_url}/api/identity/register", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Identity successfully registered![/green]")
            console.print(f"Party ID: {party_id}")
            console.print(f"Claim: {claim}")
            console.print(f"ZK Proof: {data.get('zk_proof', 'N/A')}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@identity_group.command(name="verify")
@click.option('--party-id', required=True, help='Unique identifier for the party')
@click.option('--zk-proof', required=True, help='ZK proof to verify')
def verify_identity(party_id, zk_proof):
    """Verify an identity using ZK proof"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'party_id': party_id,
            'zk_proof': zk_proof
        }
        
        response = make_api_request(f"{api_url}/api/identity/verify", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            if data.get('verified', False):
                console.print(f"[green]Identity verified successfully![/green]")
                console.print(f"Party ID: {party_id}")
                console.print(f"Claim: {data.get('claim', 'N/A')}")
            else:
                console.print(f"[red]Identity verification failed![/red]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@identity_group.command(name="list")
@click.option('--claim', help='Filter by claim type')
def list_identities(claim):
    """List registered identities"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {}
        if claim:
            params['claim'] = claim
            
        response = make_api_request(f"{api_url}/api/identity/list", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            identities = data.get('identities', [])
            
            if not identities:
                console.print("[yellow]No identities found.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Party ID")
            table.add_column("Claim")
            table.add_column("Name")
            table.add_column("Created At")
            
            for identity in identities:
                metadata = identity.get('metadata', {})
                name = metadata.get('name', 'N/A')
                table.add_row(
                    identity.get('party_id', 'N/A'),
                    identity.get('claim', 'N/A'),
                    name,
                    identity.get('created_at', 'N/A')
                )
            
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@identity_group.command(name="get")
@click.option('--party-id', required=True, help='Unique identifier for the party')
def get_identity(party_id):
    """Get details of a specific identity"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/identity/{party_id}", method="GET")
        
        if response.ok:
            identity = response.json()
            
            console.print(f"[bold]Identity Details:[/bold]")
            console.print(f"Party ID: {identity.get('party_id', 'N/A')}")
            console.print(f"Claim: {identity.get('claim', 'N/A')}")
            console.print(f"ZK Proof: {identity.get('zk_proof', 'N/A')}")
            console.print(f"Created At: {identity.get('created_at', 'N/A')}")
            
            metadata = identity.get('metadata', {})
            if metadata:
                console.print(f"\n[bold]Metadata:[/bold]")
                for key, value in metadata.items():
                    console.print(f"{key}: {value}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")
