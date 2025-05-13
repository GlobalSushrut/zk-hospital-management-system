"""
Consent commands for ZK Health CLI
"""

import os
import json
import click
from rich.console import Console
from rich.table import Table
from rich.panel import Panel

from utils import make_api_request, handle_api_error

console = Console()

@click.group(name="consent")
def consent_group():
    """Manage consent agreements"""
    pass

@consent_group.command(name="create")
@click.option('--patient-id', required=True, help='ID of the patient giving consent')
@click.option('--type', 'consent_type', required=True, 
              type=click.Choice(['treatment', 'data_sharing', 'research', 'emergency']),
              help='Type of consent')
@click.option('--description', required=True, help='Description of the consent agreement')
@click.option('--party-ids', required=True, help='Comma-separated list of party IDs')
@click.option('--roles', required=True, help='Comma-separated list of roles (same order as party IDs)')
@click.option('--expiry-days', required=True, type=int, help='Number of days until consent expires')
@click.option('--all-required/--any-required', default=True, 
              help='Whether all parties need to approve or just any')
@click.option('--resources', help='Comma-separated list of resources this consent applies to')
def create_consent(patient_id, consent_type, description, party_ids, roles, 
                  expiry_days, all_required, resources):
    """Create a new multi-party consent agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        party_id_list = [id.strip() for id in party_ids.split(',')]
        role_list = [role.strip() for role in roles.split(',')]
        
        if len(party_id_list) != len(role_list):
            console.print("[red]Error: Number of party IDs and roles must be the same[/red]")
            return
            
        resource_list = []
        if resources:
            resource_list = [resource.strip() for resource in resources.split(',')]
            
        payload = {
            'patient_id': patient_id,
            'consent_type': consent_type,
            'description': description,
            'party_ids': party_id_list,
            'roles': role_list,
            'expiry_days': expiry_days,
            'all_parties_required': all_required,
            'resources': resource_list
        }
        
        response = make_api_request(f"{api_url}/api/consent", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Consent agreement created successfully![/green]")
            console.print(f"Consent ID: {data.get('consent_id', 'N/A')}")
            console.print(f"Status: {data.get('status', 'pending')}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@consent_group.command(name="approve")
@click.option('--consent-id', required=True, help='ID of the consent agreement')
@click.option('--party-id', required=True, help='ID of the party approving')
@click.option('--zk-proof', required=True, help='ZK proof of the party')
def approve_consent(consent_id, party_id, zk_proof):
    """Approve a consent agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'consent_id': consent_id,
            'party_id': party_id,
            'zk_proof': zk_proof
        }
        
        response = make_api_request(f"{api_url}/api/consent/approve", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Consent agreement approved successfully![/green]")
            console.print(f"Consent ID: {consent_id}")
            console.print(f"New Status: {data.get('status', 'unknown')}")
            
            # Show if the consent is now active
            if data.get('status') == 'active':
                console.print(f"[green bold]Consent is now ACTIVE and can be used![/green bold]")
            else:
                console.print(f"[yellow]Waiting for other parties to approve...[/yellow]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@consent_group.command(name="revoke")
@click.option('--consent-id', required=True, help='ID of the consent agreement')
@click.option('--party-id', required=True, help='ID of the party revoking')
@click.option('--zk-proof', required=True, help='ZK proof of the party')
@click.option('--reason', required=True, help='Reason for revocation')
def revoke_consent(consent_id, party_id, zk_proof, reason):
    """Revoke a consent agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'consent_id': consent_id,
            'party_id': party_id,
            'zk_proof': zk_proof,
            'reason': reason
        }
        
        response = make_api_request(f"{api_url}/api/consent/revoke", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[yellow]Consent agreement revoked successfully![/yellow]")
            console.print(f"Consent ID: {consent_id}")
            console.print(f"New Status: {data.get('status', 'revoked')}")
            console.print(f"Revoked By: {party_id}")
            console.print(f"Reason: {reason}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@consent_group.command(name="list")
@click.option('--patient-id', help='Filter by patient ID')
@click.option('--party-id', help='Filter by party ID (participant)')
@click.option('--status', type=click.Choice(['pending', 'active', 'expired', 'revoked']), 
              help='Filter by consent status')
def list_consents(patient_id, party_id, status):
    """List consent agreements"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {}
        if patient_id:
            params['patient_id'] = patient_id
        if party_id:
            params['party_id'] = party_id
        if status:
            params['status'] = status
            
        response = make_api_request(f"{api_url}/api/consent/list", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            consents = data.get('consents', [])
            
            if not consents:
                console.print("[yellow]No consent agreements found.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Consent ID")
            table.add_column("Patient ID")
            table.add_column("Type")
            table.add_column("Status")
            table.add_column("Created")
            table.add_column("Expires")
            
            for consent in consents:
                status_color = {
                    'pending': 'yellow',
                    'active': 'green',
                    'expired': 'dim',
                    'revoked': 'red'
                }.get(consent.get('status', ''), 'white')
                
                table.add_row(
                    consent.get('consent_id', 'N/A'),
                    consent.get('patient_id', 'N/A'),
                    consent.get('consent_type', 'N/A'),
                    f"[{status_color}]{consent.get('status', 'N/A')}[/{status_color}]",
                    consent.get('created_at', 'N/A'),
                    consent.get('expiry_date', 'N/A')
                )
            
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@consent_group.command(name="get")
@click.option('--consent-id', required=True, help='ID of the consent agreement')
def get_consent(consent_id):
    """Get details of a specific consent agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/consent/{consent_id}", method="GET")
        
        if response.ok:
            consent = response.json()
            
            status_color = {
                'pending': 'yellow',
                'active': 'green',
                'expired': 'dim',
                'revoked': 'red'
            }.get(consent.get('status', ''), 'white')
            
            console.print(Panel.fit(
                f"[bold]Consent ID:[/bold] {consent.get('consent_id', 'N/A')}\n"
                f"[bold]Type:[/bold] {consent.get('consent_type', 'N/A')}\n"
                f"[bold]Status:[/bold] [{status_color}]{consent.get('status', 'N/A')}[/{status_color}]\n"
                f"[bold]Patient ID:[/bold] {consent.get('patient_id', 'N/A')}\n"
                f"[bold]Description:[/bold] {consent.get('description', 'N/A')}\n"
                f"[bold]Created:[/bold] {consent.get('created_at', 'N/A')}\n"
                f"[bold]Expires:[/bold] {consent.get('expiry_date', 'N/A')}\n"
                f"[bold]All Parties Required:[/bold] {'Yes' if consent.get('all_parties_required', True) else 'No'}",
                title="Consent Agreement", border_style="blue"
            ))
            
            # Show parties and their approval status
            parties = consent.get('parties', [])
            if parties:
                party_table = Table(show_header=True, header_style="bold blue")
                party_table.add_column("Party ID")
                party_table.add_column("Role")
                party_table.add_column("Status")
                party_table.add_column("Approved/Revoked At")
                
                for party in parties:
                    status = party.get('status', 'pending')
                    status_color = {
                        'pending': 'yellow',
                        'approved': 'green',
                        'revoked': 'red'
                    }.get(status, 'white')
                    
                    party_table.add_row(
                        party.get('party_id', 'N/A'),
                        party.get('role', 'N/A'),
                        f"[{status_color}]{status}[/{status_color}]",
                        party.get('timestamp', 'N/A')
                    )
                
                console.print("\n[bold]Parties:[/bold]")
                console.print(party_table)
            
            # Show resources
            resources = consent.get('resources', [])
            if resources:
                console.print("\n[bold]Resources:[/bold]")
                for resource in resources:
                    console.print(f"- {resource}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@consent_group.command(name="verify")
@click.option('--consent-id', required=True, help='ID of the consent agreement')
@click.option('--party-id', required=True, help='ID of the party to verify access for')
@click.option('--resource', help='Specific resource to verify access to')
def verify_consent(consent_id, party_id, resource):
    """Verify if a party has consent to access a resource"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'party_id': party_id
        }
        
        if resource:
            params['resource'] = resource
            
        response = make_api_request(f"{api_url}/api/consent/{consent_id}/verify", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            
            if data.get('has_consent', False):
                console.print(f"[green]✓ Party has valid consent access![/green]")
                
                if resource:
                    console.print(f"Resource: {resource}")
                    
                console.print(f"Consent ID: {consent_id}")
                console.print(f"Party ID: {party_id}")
                console.print(f"Role: {data.get('role', 'N/A')}")
            else:
                console.print(f"[red]✗ Party does NOT have valid consent access![/red]")
                console.print(f"Reason: {data.get('reason', 'Unknown')}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")
