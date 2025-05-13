"""
Oracle Chain Validator commands for ZK Health CLI
"""

import os
import json
import click
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.syntax import Syntax

from utils import make_api_request, handle_api_error

console = Console()

@click.group(name="oracle")
def oracle_group():
    """Manage Oracle Chain Validator agreements"""
    pass

@oracle_group.command(name="create")
@click.option('--name', required=True, help='Name of the agreement')
@click.option('--description', required=True, help='Description of the agreement')
@click.option('--jurisdiction', required=True, help='Legal jurisdiction code (e.g., US-HIPAA, EU-GDPR)')
@click.option('--agreement-file', required=True, type=click.Path(exists=True), 
              help='Path to JSON file containing agreement clauses')
def create_agreement(name, description, jurisdiction, agreement_file):
    """Create a new Oracle agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        # Read agreement file
        with open(agreement_file, 'r') as f:
            agreement_data = json.load(f)
        
        # Validate agreement structure
        if 'clauses' not in agreement_data:
            console.print("[red]Error: Agreement file must contain 'clauses' array[/red]")
            return
            
        payload = {
            'name': name,
            'description': description,
            'jurisdiction': jurisdiction,
            'clauses': agreement_data['clauses']
        }
        
        response = make_api_request(f"{api_url}/api/oracle/agreement", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Oracle agreement created successfully![/green]")
            console.print(f"Agreement ID: {data.get('agreement_id', 'N/A')}")
            console.print(f"Hash: {data.get('agreement_hash', 'N/A')}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@oracle_group.command(name="list")
@click.option('--jurisdiction', help='Filter by jurisdiction')
def list_agreements(jurisdiction):
    """List Oracle agreements"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {}
        if jurisdiction:
            params['jurisdiction'] = jurisdiction
            
        response = make_api_request(f"{api_url}/api/oracle/agreements", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            agreements = data.get('agreements', [])
            
            if not agreements:
                console.print("[yellow]No Oracle agreements found.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Agreement ID")
            table.add_column("Name")
            table.add_column("Jurisdiction")
            table.add_column("Hash")
            table.add_column("Created At")
            
            for agreement in agreements:
                table.add_row(
                    agreement.get('agreement_id', 'N/A'),
                    agreement.get('name', 'N/A'),
                    agreement.get('jurisdiction', 'N/A'),
                    agreement.get('agreement_hash', 'N/A')[:10] + '...',
                    agreement.get('created_at', 'N/A')
                )
            
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@oracle_group.command(name="get")
@click.option('--agreement-id', required=True, help='ID of the Oracle agreement')
def get_agreement(agreement_id):
    """Get details of a specific Oracle agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/oracle/agreement/{agreement_id}", method="GET")
        
        if response.ok:
            agreement = response.json()
            
            console.print(Panel.fit(
                f"[bold]Agreement ID:[/bold] {agreement.get('agreement_id', 'N/A')}\n"
                f"[bold]Name:[/bold] {agreement.get('name', 'N/A')}\n"
                f"[bold]Description:[/bold] {agreement.get('description', 'N/A')}\n"
                f"[bold]Jurisdiction:[/bold] {agreement.get('jurisdiction', 'N/A')}\n"
                f"[bold]Hash:[/bold] {agreement.get('agreement_hash', 'N/A')}\n"
                f"[bold]Created:[/bold] {agreement.get('created_at', 'N/A')}",
                title="Oracle Agreement", border_style="blue"
            ))
            
            # Show clauses
            clauses = agreement.get('clauses', [])
            if clauses:
                console.print("\n[bold]Clauses:[/bold]")
                
                clause_table = Table(show_header=True, header_style="bold blue")
                clause_table.add_column("ID")
                clause_table.add_column("Title")
                clause_table.add_column("Type")
                
                for clause in clauses:
                    clause_table.add_row(
                        clause.get('clause_id', 'N/A'),
                        clause.get('title', 'N/A'),
                        clause.get('type', 'N/A')
                    )
                
                console.print(clause_table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@oracle_group.command(name="get-clause")
@click.option('--agreement-id', required=True, help='ID of the Oracle agreement')
@click.option('--clause-id', required=True, help='ID of the clause')
def get_clause(agreement_id, clause_id):
    """Get details of a specific clause in an Oracle agreement"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(
            f"{api_url}/api/oracle/agreement/{agreement_id}/clause/{clause_id}", 
            method="GET"
        )
        
        if response.ok:
            clause = response.json()
            
            console.print(Panel.fit(
                f"[bold]Clause ID:[/bold] {clause.get('clause_id', 'N/A')}\n"
                f"[bold]Title:[/bold] {clause.get('title', 'N/A')}\n"
                f"[bold]Type:[/bold] {clause.get('type', 'N/A')}\n"
                f"[bold]Description:[/bold] {clause.get('description', 'N/A')}",
                title="Clause Details", border_style="blue"
            ))
            
            # Show preconditions
            preconditions = clause.get('preconditions', {})
            if preconditions:
                console.print("\n[bold]Preconditions:[/bold]")
                preconditions_json = json.dumps(preconditions, indent=2)
                console.print(Syntax(preconditions_json, "json", theme="monokai", line_numbers=True))
            
            # Show execute conditions
            execute = clause.get('execute', {})
            if execute:
                console.print("\n[bold]Execute Conditions:[/bold]")
                execute_json = json.dumps(execute, indent=2)
                console.print(Syntax(execute_json, "json", theme="monokai", line_numbers=True))
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@oracle_group.command(name="validate-event")
@click.option('--agreement-id', required=True, help='ID of the Oracle agreement')
@click.option('--event-id', required=True, help='ID of the event to validate')
@click.option('--event-type', required=True, help='Type of event')
@click.option('--clause-ids', required=True, help='Comma-separated list of clause IDs to validate')
@click.option('--signer-id', required=True, help='ID of the event signer')
@click.option('--zk-proof', required=True, help='ZK proof of the signer')
@click.option('--context-file', required=True, type=click.Path(exists=True), 
              help='Path to JSON file containing event context')
def validate_event(agreement_id, event_id, event_type, clause_ids, signer_id, zk_proof, context_file):
    """Validate an execution event against oracle agreement clauses"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        # Read context file
        with open(context_file, 'r') as f:
            context_data = json.load(f)
        
        clause_id_list = [id.strip() for id in clause_ids.split(',')]
        
        payload = {
            'event_id': event_id,
            'event_type': event_type,
            'agreement_id': agreement_id,
            'clause_ids': clause_id_list,
            'signer_id': signer_id,
            'zk_proof': zk_proof,
            'context': context_data
        }
        
        response = make_api_request(f"{api_url}/api/oracle/validate", method="POST", json=payload)
        
        if response.ok:
            result = response.json()
            
            if result.get('valid', False):
                console.print(f"[green]✓ Event validation SUCCESSFUL![/green]")
            else:
                console.print(f"[red]✗ Event validation FAILED![/red]")
                
            console.print(f"[bold]Event ID:[/bold] {result.get('event_id', event_id)}")
            console.print(f"[bold]Agreement ID:[/bold] {result.get('agreement_id', agreement_id)}")
            console.print(f"[bold]Validation Time:[/bold] {result.get('timestamp', 'N/A')}")
            
            # Show clause validation results
            clause_validations = result.get('clause_validations', {})
            if clause_validations:
                console.print("\n[bold]Clause Validations:[/bold]")
                
                clause_table = Table(show_header=True, header_style="bold blue")
                clause_table.add_column("Clause ID")
                clause_table.add_column("Result")
                
                for clause_id, is_valid in clause_validations.items():
                    status_color = "green" if is_valid else "red"
                    status_symbol = "✓" if is_valid else "✗"
                    
                    clause_table.add_row(
                        clause_id,
                        f"[{status_color}]{status_symbol} {'PASSED' if is_valid else 'FAILED'}[/{status_color}]"
                    )
                
                console.print(clause_table)
            
            # Show validation notes
            validation_notes = result.get('validation_notes', [])
            if validation_notes:
                console.print("\n[bold]Validation Notes:[/bold]")
                for note in validation_notes:
                    console.print(f"- {note}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@oracle_group.command(name="create-example")
@click.option('--output-file', required=True, type=click.Path(), 
              help='Path to save the example agreement JSON file')
@click.option('--type', 'example_type', required=True,
              type=click.Choice(['hipaa', 'gdpr', 'telemedicine']),
              help='Type of example agreement to create')
def create_example(output_file, example_type):
    """Create an example Oracle agreement template"""
    try:
        example = {}
        
        if example_type == 'hipaa':
            example = {
                "clauses": [
                    {
                        "clause_id": "hipaa-phi-access",
                        "title": "PHI Access Control",
                        "type": "compliance",
                        "description": "Controls access to Protected Health Information (PHI)",
                        "preconditions": {
                            "actor_claim": ["doctor", "nurse", "admin"],
                            "patient_consent": True,
                            "emergency_override": False
                        },
                        "execute": {
                            "log_access": True,
                            "restrict_fields": ["ssn", "financial"],
                            "audit_trail": True
                        }
                    },
                    {
                        "clause_id": "hipaa-minimum-necessary",
                        "title": "Minimum Necessary Rule",
                        "type": "compliance",
                        "description": "Ensures only minimum necessary PHI is accessed",
                        "preconditions": {
                            "purpose_specified": True,
                            "scope_limited": True
                        },
                        "execute": {
                            "filter_data": True,
                            "log_purpose": True
                        }
                    }
                ]
            }
        elif example_type == 'gdpr':
            example = {
                "clauses": [
                    {
                        "clause_id": "gdpr-data-processing",
                        "title": "Lawful Data Processing",
                        "type": "compliance",
                        "description": "Ensures data processing follows GDPR principles",
                        "preconditions": {
                            "explicit_consent": True,
                            "purpose_specified": True,
                            "data_minimization": True
                        },
                        "execute": {
                            "record_processing": True,
                            "notify_subject": True
                        }
                    },
                    {
                        "clause_id": "gdpr-right-to-access",
                        "title": "Right to Access",
                        "type": "compliance",
                        "description": "Implements the data subject's right to access their data",
                        "preconditions": {
                            "identity_verified": True,
                            "request_validated": True
                        },
                        "execute": {
                            "provide_data_copy": True,
                            "include_processing_info": True,
                            "respond_within_days": 30
                        }
                    }
                ]
            }
        elif example_type == 'telemedicine':
            example = {
                "clauses": [
                    {
                        "clause_id": "telemedicine-jurisdiction",
                        "title": "Jurisdictional Compliance",
                        "type": "legal",
                        "description": "Ensures telemedicine practice complies with local laws",
                        "preconditions": {
                            "doctor_licensed_in_jurisdiction": True,
                            "patient_location_verified": True,
                            "service_allowed_in_jurisdiction": True
                        },
                        "execute": {
                            "log_jurisdictional_check": True,
                            "apply_local_regulations": True
                        }
                    },
                    {
                        "clause_id": "telemedicine-prescription",
                        "title": "Prescription Issuance",
                        "type": "medical",
                        "description": "Controls electronic prescription issuance",
                        "preconditions": {
                            "valid_consultation": True,
                            "doctor_prescription_rights": True,
                            "medication_allowed_for_telemedicine": True,
                            "patient_identity_verified": True
                        },
                        "execute": {
                            "generate_secure_prescription": True,
                            "log_prescription_details": True,
                            "notify_pharmacy": True
                        }
                    },
                    {
                        "clause_id": "telemedicine-emergency-protocol",
                        "title": "Emergency Protocol",
                        "type": "safety",
                        "description": "Defines actions in case of medical emergency during teleconsultation",
                        "preconditions": {
                            "emergency_detected": True,
                            "patient_location_known": True
                        },
                        "execute": {
                            "notify_emergency_services": True,
                            "provide_patient_data": True,
                            "document_incident": True
                        }
                    }
                ]
            }
        
        # Write to file
        with open(output_file, 'w') as f:
            json.dump(example, f, indent=2)
            
        console.print(f"[green]Example {example_type} agreement saved to {output_file}[/green]")
        console.print(f"You can use this file with the 'oracle create' command:")
        console.print(f"zk_health_cli.py oracle create --name 'Example {example_type.upper()}' --description 'Example agreement' --jurisdiction '{example_type.upper()}' --agreement-file {output_file}")
        
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")
