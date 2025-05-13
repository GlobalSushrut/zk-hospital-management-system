"""
ZK API Gateway commands for ZK Health CLI
"""

import os
import json
import click
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from datetime import datetime, timedelta

from utils import make_api_request, handle_api_error

console = Console()

@click.group(name="gateway")
def gateway_group():
    """Manage ZK API Gateway tokens and access control"""
    pass

@gateway_group.command(name="generate-token")
@click.option('--party-id', required=True, help='ID of the party requesting the token')
@click.option('--claim', required=True, help='Claim type (e.g., doctor, patient, admin)')
@click.option('--validity-hours', type=int, default=24, help='Token validity in hours')
@click.option('--zk-proof', required=True, help='ZK proof of the party')
def generate_token(party_id, claim, validity_hours, zk_proof):
    """Generate a new ZK API token"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'party_id': party_id,
            'claim': claim,
            'validity_hours': validity_hours,
            'zk_proof': zk_proof
        }
        
        response = make_api_request(f"{api_url}/api/gateway/token", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]ZK API token generated successfully![/green]")
            console.print(Panel.fit(
                f"[bold]Token ID:[/bold] {data.get('token_id', 'N/A')}\n"
                f"[bold]Party ID:[/bold] {party_id}\n"
                f"[bold]Claim:[/bold] {claim}\n"
                f"[bold]Created:[/bold] {data.get('created_at', 'N/A')}\n"
                f"[bold]Expires:[/bold] {data.get('expires_at', 'N/A')}\n",
                title="API Token", border_style="green"
            ))
            
            console.print("\n[bold]To use this token:[/bold]")
            console.print("Include it in API requests with the header:")
            console.print("[blue]X-ZK-API-Key: {token_id}[/blue]".format(token_id=data.get('token_id', 'N/A')))
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="validate-token")
@click.option('--token-id', required=True, help='ID of the token to validate')
def validate_token(token_id):
    """Validate a ZK API token"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/gateway/token/{token_id}/validate", method="GET")
        
        if response.ok:
            data = response.json()
            
            if data.get('valid', False):
                console.print(f"[green]✓ Token is valid![/green]")
                console.print(f"Party ID: {data.get('party_id', 'N/A')}")
                console.print(f"Claim: {data.get('claim', 'N/A')}")
                console.print(f"Expires: {data.get('expires_at', 'N/A')}")
            else:
                console.print(f"[red]✗ Token is invalid![/red]")
                console.print(f"Reason: {data.get('reason', 'Unknown')}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="revoke-token")
@click.option('--token-id', required=True, help='ID of the token to revoke')
@click.option('--admin-id', required=True, help='ID of the admin revoking the token')
@click.option('--zk-proof', required=True, help='ZK proof of the admin')
def revoke_token(token_id, admin_id, zk_proof):
    """Revoke a ZK API token"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'token_id': token_id,
            'admin_id': admin_id,
            'zk_proof': zk_proof
        }
        
        response = make_api_request(f"{api_url}/api/gateway/token/revoke", method="POST", json=payload)
        
        if response.ok:
            console.print(f"[yellow]Token {token_id} has been revoked successfully![/yellow]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="list-tokens")
@click.option('--party-id', required=True, help='ID of the party to list tokens for')
@click.option('--admin-id', help='Admin ID for viewing other user tokens')
@click.option('--zk-proof', required=True, help='ZK proof of the requester')
def list_tokens(party_id, admin_id, zk_proof):
    """List active ZK API tokens for a party"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'party_id': party_id,
            'zk_proof': zk_proof
        }
        
        if admin_id:
            params['admin_id'] = admin_id
            
        response = make_api_request(f"{api_url}/api/gateway/tokens", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            tokens = data.get('tokens', [])
            
            if not tokens:
                console.print(f"[yellow]No active tokens found for party {party_id}.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Token ID")
            table.add_column("Claim")
            table.add_column("Created")
            table.add_column("Expires")
            table.add_column("Last Used")
            
            now = datetime.now()
            
            for token in tokens:
                # Parse expiry to calculate time remaining
                expiry = datetime.fromisoformat(token.get('expires_at', '').replace('Z', '+00:00'))
                time_left = expiry - now
                
                expires_text = token.get('expires_at', 'N/A')
                if time_left.total_seconds() > 0:
                    days = time_left.days
                    hours = time_left.seconds // 3600
                    mins = (time_left.seconds % 3600) // 60
                    
                    if days > 0:
                        expires_text += f" ({days}d {hours}h left)"
                    else:
                        expires_text += f" ({hours}h {mins}m left)"
                
                table.add_row(
                    token.get('token_id', 'N/A'),
                    token.get('claim', 'N/A'),
                    token.get('created_at', 'N/A'),
                    expires_text,
                    token.get('last_used_at', 'N/A')
                )
            
            console.print(f"[bold]Active tokens for party {party_id}:[/bold]")
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="add-rate-limit")
@click.option('--admin-id', required=True, help='ID of the admin adding the rate limit')
@click.option('--zk-proof', required=True, help='ZK proof of the admin')
@click.option('--endpoint', required=True, help='API endpoint pattern (e.g., /api/treatment/*)')
@click.option('--requests-per-minute', type=int, required=True, help='Maximum requests per minute')
@click.option('--requests-per-hour', type=int, required=True, help='Maximum requests per hour')
@click.option('--requests-per-day', type=int, required=True, help='Maximum requests per day')
@click.option('--applies-to', required=True, help='Comma-separated list of claims this applies to (or * for all)')
@click.option('--block-duration', type=int, default=15, help='Blocking duration in minutes')
def add_rate_limit(admin_id, zk_proof, endpoint, requests_per_minute, 
                 requests_per_hour, requests_per_day, applies_to, block_duration):
    """Add a new rate limit rule"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        applies_to_list = [claim.strip() for claim in applies_to.split(',')]
        
        payload = {
            'admin_id': admin_id,
            'zk_proof': zk_proof,
            'endpoint': endpoint,
            'requests_per_minute': requests_per_minute,
            'requests_per_hour': requests_per_hour,
            'requests_per_day': requests_per_day,
            'applies_to': applies_to_list,
            'block_duration': block_duration
        }
        
        response = make_api_request(f"{api_url}/api/gateway/ratelimit", method="POST", json=payload)
        
        if response.ok:
            console.print(f"[green]Rate limit rule added successfully![/green]")
            console.print(f"Endpoint: {endpoint}")
            console.print(f"Limits: {requests_per_minute}/min, {requests_per_hour}/hour, {requests_per_day}/day")
            console.print(f"Applies to: {applies_to}")
            console.print(f"Block duration: {block_duration} minutes")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="list-rate-limits")
@click.option('--admin-id', required=True, help='ID of the admin viewing rate limits')
@click.option('--zk-proof', required=True, help='ZK proof of the admin')
def list_rate_limits(admin_id, zk_proof):
    """List rate limit rules"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'admin_id': admin_id,
            'zk_proof': zk_proof
        }
            
        response = make_api_request(f"{api_url}/api/gateway/ratelimits", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            rules = data.get('rules', [])
            
            if not rules:
                console.print(f"[yellow]No rate limit rules found.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Endpoint")
            table.add_column("Per Minute")
            table.add_column("Per Hour")
            table.add_column("Per Day")
            table.add_column("Applies To")
            table.add_column("Block Duration")
            
            for rule in rules:
                applies_to = ", ".join(rule.get('applies_to', []))
                if not applies_to:
                    applies_to = "*"
                    
                table.add_row(
                    rule.get('endpoint', 'N/A'),
                    str(rule.get('requests_per_minute', 'N/A')),
                    str(rule.get('requests_per_hour', 'N/A')),
                    str(rule.get('requests_per_day', 'N/A')),
                    applies_to,
                    f"{rule.get('block_duration', 'N/A')} min"
                )
            
            console.print(f"[bold]Rate Limit Rules:[/bold]")
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="delete-rate-limit")
@click.option('--admin-id', required=True, help='ID of the admin deleting the rate limit')
@click.option('--zk-proof', required=True, help='ZK proof of the admin')
@click.option('--endpoint', required=True, help='API endpoint pattern to delete')
def delete_rate_limit(admin_id, zk_proof, endpoint):
    """Delete a rate limit rule"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'admin_id': admin_id,
            'zk_proof': zk_proof,
            'endpoint': endpoint
        }
            
        response = make_api_request(f"{api_url}/api/gateway/ratelimit", method="DELETE", params=params)
        
        if response.ok:
            console.print(f"[yellow]Rate limit rule for endpoint {endpoint} deleted successfully![/yellow]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@gateway_group.command(name="unblock-party")
@click.option('--admin-id', required=True, help='ID of the admin unblocking the party')
@click.option('--zk-proof', required=True, help='ZK proof of the admin')
@click.option('--party-id', required=True, help='ID of the party to unblock')
def unblock_party(admin_id, zk_proof, party_id):
    """Unblock a party that has been rate limited"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'admin_id': admin_id,
            'zk_proof': zk_proof,
            'party_id': party_id
        }
        
        response = make_api_request(f"{api_url}/api/gateway/unblock", method="POST", json=payload)
        
        if response.ok:
            console.print(f"[green]Party {party_id} has been unblocked successfully![/green]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")
