"""
Document commands for ZK Health CLI
"""

import os
import json
import click
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.progress import Progress

from utils import make_api_request, handle_api_error

console = Console()

@click.group(name="document")
def document_group():
    """Manage secure medical documents"""
    pass

@document_group.command(name="upload")
@click.option('--patient-id', required=True, help='ID of the patient the document belongs to')
@click.option('--uploader-id', required=True, help='ID of the party uploading the document')
@click.option('--doc-type', required=True, 
              type=click.Choice(['medical_record', 'lab_result', 'prescription', 'image', 'scan']),
              help='Type of document')
@click.option('--description', required=True, help='Description of the document')
@click.option('--consent-id', required=True, help='ID of the consent agreement authorizing this upload')
@click.option('--file-path', required=True, type=click.Path(exists=True), 
              help='Path to the file to upload')
@click.option('--metadata', help='Additional metadata as JSON string')
@click.option('--zk-proof', required=True, help='ZK proof of the uploader')
def upload_document(patient_id, uploader_id, doc_type, description, consent_id, 
                   file_path, metadata, zk_proof):
    """Upload a new document to the secure archive"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        meta_data = {}
        if metadata:
            meta_data = json.loads(metadata)
            
        # Get file size for progress reporting
        file_size = os.path.getsize(file_path)
        
        # Prepare multipart form data
        with open(file_path, 'rb') as f:
            files = {'file': (os.path.basename(file_path), f)}
            
            payload = {
                'patient_id': patient_id,
                'uploader_id': uploader_id,
                'doc_type': doc_type,
                'description': description,
                'consent_id': consent_id,
                'metadata': json.dumps(meta_data),
                'zk_proof': zk_proof
            }
            
            with Progress() as progress:
                upload_task = progress.add_task("[green]Uploading document...", total=100)
                
                # Custom callback for upload progress
                def upload_progress(monitor):
                    progress.update(upload_task, completed=int(100 * monitor.bytes_read / file_size))
                
                response = make_api_request(
                    f"{api_url}/api/document/upload", 
                    method="POST", 
                    data=payload,
                    files=files,
                    progress_callback=upload_progress
                )
            
            if response.ok:
                data = response.json()
                console.print(f"[green]Document uploaded successfully![/green]")
                console.print(f"Document ID: {data.get('document_id', 'N/A')}")
                console.print(f"Merkle Root: {data.get('merkle_root', 'N/A')}")
                console.print(f"File Hash: {data.get('file_hash', 'N/A')}")
            else:
                handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@document_group.command(name="verify")
@click.option('--document-id', required=True, help='ID of the document to verify')
def verify_document(document_id):
    """Verify the integrity of a document"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/document/{document_id}/verify", method="GET")
        
        if response.ok:
            data = response.json()
            
            if data.get('verified', False):
                console.print(f"[green]✓ Document integrity verified![/green]")
                console.print(f"Document ID: {document_id}")
                console.print(f"Merkle Root: {data.get('merkle_root', 'N/A')}")
                console.print(f"Verification Time: {data.get('verification_time', 'N/A')}")
            else:
                console.print(f"[red]✗ Document verification FAILED![/red]")
                console.print(f"Reason: {data.get('reason', 'Unknown')}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@document_group.command(name="list")
@click.option('--patient-id', help='Filter by patient ID')
@click.option('--doc-type', help='Filter by document type')
@click.option('--uploader-id', help='Filter by uploader ID')
def list_documents(patient_id, doc_type, uploader_id):
    """List documents in the archive"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {}
        if patient_id:
            params['patient_id'] = patient_id
        if doc_type:
            params['doc_type'] = doc_type
        if uploader_id:
            params['uploader_id'] = uploader_id
            
        response = make_api_request(f"{api_url}/api/document/list", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            documents = data.get('documents', [])
            
            if not documents:
                console.print("[yellow]No documents found.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Document ID")
            table.add_column("Patient ID")
            table.add_column("Type")
            table.add_column("Description")
            table.add_column("Uploaded By")
            table.add_column("Uploaded At")
            
            for doc in documents:
                table.add_row(
                    doc.get('document_id', 'N/A'),
                    doc.get('patient_id', 'N/A'),
                    doc.get('doc_type', 'N/A'),
                    doc.get('description', 'N/A'),
                    doc.get('uploader_id', 'N/A'),
                    doc.get('uploaded_at', 'N/A')
                )
            
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@document_group.command(name="get")
@click.option('--document-id', required=True, help='ID of the document')
def get_document(document_id):
    """Get details of a document"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/document/{document_id}", method="GET")
        
        if response.ok:
            doc = response.json()
            
            console.print(Panel.fit(
                f"[bold]Document ID:[/bold] {doc.get('document_id', 'N/A')}\n"
                f"[bold]Patient ID:[/bold] {doc.get('patient_id', 'N/A')}\n"
                f"[bold]Type:[/bold] {doc.get('doc_type', 'N/A')}\n"
                f"[bold]Description:[/bold] {doc.get('description', 'N/A')}\n"
                f"[bold]Uploader ID:[/bold] {doc.get('uploader_id', 'N/A')}\n"
                f"[bold]Uploaded At:[/bold] {doc.get('uploaded_at', 'N/A')}\n"
                f"[bold]File Name:[/bold] {doc.get('file_name', 'N/A')}\n"
                f"[bold]File Size:[/bold] {doc.get('file_size', 'N/A')} bytes\n"
                f"[bold]File Hash:[/bold] {doc.get('file_hash', 'N/A')}\n"
                f"[bold]Merkle Root:[/bold] {doc.get('merkle_root', 'N/A')}\n"
                f"[bold]Consent ID:[/bold] {doc.get('consent_id', 'N/A')}",
                title="Document Details", border_style="blue"
            ))
            
            # Show metadata if available
            metadata = doc.get('metadata', {})
            if metadata:
                console.print("\n[bold]Metadata:[/bold]")
                for key, value in metadata.items():
                    console.print(f"{key}: {value}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@document_group.command(name="download")
@click.option('--document-id', required=True, help='ID of the document to download')
@click.option('--requester-id', required=True, help='ID of the party requesting the download')
@click.option('--zk-proof', required=True, help='ZK proof of the requester')
@click.option('--output-dir', required=True, type=click.Path(exists=True), 
              help='Directory to save the downloaded file')
def download_document(document_id, requester_id, zk_proof, output_dir):
    """Download a document from the archive"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'requester_id': requester_id,
            'zk_proof': zk_proof
        }
        
        with Progress() as progress:
            download_task = progress.add_task("[green]Downloading document...", total=100)
            
            # Custom callback for download progress
            def download_progress(monitor):
                if monitor.total_bytes:
                    progress.update(download_task, completed=int(100 * monitor.bytes_read / monitor.total_bytes))
            
            response = make_api_request(
                f"{api_url}/api/document/{document_id}/download", 
                method="GET", 
                params=params,
                stream=True,
                progress_callback=download_progress
            )
        
        if response.ok:
            # Get filename from Content-Disposition header
            content_disposition = response.headers.get('Content-Disposition', '')
            filename = 'downloaded_file'
            if 'filename=' in content_disposition:
                filename = content_disposition.split('filename=')[1].strip('"')
            
            output_path = os.path.join(output_dir, filename)
            
            # Save the file
            with open(output_path, 'wb') as f:
                for chunk in response.iter_content(chunk_size=8192):
                    if chunk:
                        f.write(chunk)
            
            console.print(f"[green]Document downloaded successfully![/green]")
            console.print(f"Saved to: {output_path}")
            
            # Verify the downloaded file
            console.print("[yellow]Verifying document integrity...[/yellow]")
            
            verify_response = make_api_request(f"{api_url}/api/document/{document_id}/verify", method="GET")
            
            if verify_response.ok and verify_response.json().get('verified', False):
                console.print(f"[green]✓ Downloaded document integrity verified![/green]")
            else:
                console.print(f"[red]⚠ Warning: Could not verify document integrity![/red]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@document_group.command(name="audit")
@click.option('--document-id', required=True, help='ID of the document to audit')
@click.option('--requester-id', required=True, help='ID of the party requesting the audit')
@click.option('--zk-proof', required=True, help='ZK proof of the requester')
def audit_document(document_id, requester_id, zk_proof):
    """Get the access audit trail for a document"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'requester_id': requester_id,
            'zk_proof': zk_proof
        }
        
        response = make_api_request(
            f"{api_url}/api/document/{document_id}/audit", 
            method="GET", 
            params=params
        )
        
        if response.ok:
            data = response.json()
            audit_events = data.get('audit_events', [])
            
            if not audit_events:
                console.print("[yellow]No audit events found for this document.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Timestamp")
            table.add_column("Action")
            table.add_column("Party ID")
            table.add_column("IP Address")
            table.add_column("Details")
            
            for event in audit_events:
                table.add_row(
                    event.get('timestamp', 'N/A'),
                    event.get('action', 'N/A'),
                    event.get('party_id', 'N/A'),
                    event.get('ip_address', 'N/A'),
                    event.get('details', 'N/A')
                )
            
            console.print(f"[bold]Audit trail for document {document_id}:[/bold]")
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")
