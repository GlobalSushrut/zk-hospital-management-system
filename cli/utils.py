"""
Utility functions for ZK Health CLI
"""

import os
import json
import requests
from rich.console import Console

console = Console()

def make_api_request(url, method="GET", params=None, data=None, json=None, files=None, 
                    stream=False, progress_callback=None):
    """
    Make an API request with proper error handling
    
    Args:
        url: API endpoint URL
        method: HTTP method (GET, POST, PUT, DELETE)
        params: URL parameters
        data: Form data
        json: JSON payload
        files: Multipart files
        stream: Whether to stream the response
        progress_callback: Optional callback for upload/download progress
        
    Returns:
        Response object
    """
    
    # Add authentication token if available in environment
    headers = {}
    api_token = os.getenv('ZK_API_TOKEN')
    if api_token:
        headers['X-ZK-API-Key'] = api_token
    
    # Make the request
    try:
        if progress_callback and files:
            # Custom implementation for upload progress
            return requests.request(
                method, 
                url, 
                params=params, 
                data=data, 
                json=json,
                files=files, 
                headers=headers,
                stream=stream
            )
        elif progress_callback and stream:
            # Download with progress
            response = requests.request(
                method, 
                url, 
                params=params, 
                data=data, 
                json=json,
                files=files, 
                headers=headers,
                stream=True
            )
            
            # Create a wrapper for progress tracking
            class ProgressTracker:
                def __init__(self, response):
                    self.response = response
                    self.bytes_read = 0
                    self.total_bytes = int(response.headers.get('content-length', 0))
                    
                def iter_content(self, chunk_size=8192):
                    for chunk in self.response.iter_content(chunk_size=chunk_size):
                        self.bytes_read += len(chunk)
                        progress_callback(self)
                        yield chunk
            
            wrapper = ProgressTracker(response)
            response.iter_content = wrapper.iter_content
            return response
        else:
            return requests.request(
                method, 
                url, 
                params=params, 
                data=data, 
                json=json,
                files=files, 
                headers=headers,
                stream=stream
            )
    except requests.exceptions.ConnectionError:
        console.print("[red]Error: Could not connect to the API. Is the server running?[/red]")
        raise
    except requests.exceptions.Timeout:
        console.print("[red]Error: Request timed out.[/red]")
        raise
    except requests.exceptions.RequestException as e:
        console.print(f"[red]Error: {str(e)}[/red]")
        raise

def handle_api_error(response):
    """
    Handle API error responses
    
    Args:
        response: Response object
    """
    try:
        error_data = response.json()
        error_message = error_data.get('message', error_data.get('error', 'Unknown error'))
        console.print(f"[red]API Error ({response.status_code}): {error_message}[/red]")
    except:
        console.print(f"[red]API Error ({response.status_code}): {response.text}[/red]")

def check_service_status(url):
    """
    Check if a service is online
    
    Args:
        url: Service URL to check
        
    Returns:
        (status, details) tuple
    """
    try:
        response = requests.get(url, timeout=5)
        if response.ok:
            try:
                data = response.json()
                return True, data.get('status', 'Online')
            except:
                return True, "Online"
        else:
            return False, f"Status code: {response.status_code}"
    except:
        return False, "Connection failed"

def format_timestamp(timestamp):
    """
    Format a timestamp for display
    
    Args:
        timestamp: ISO timestamp
        
    Returns:
        Formatted timestamp string
    """
    if not timestamp:
        return "N/A"
    
    # Remove the 'Z' suffix and microseconds if present
    timestamp = timestamp.replace('Z', '')
    if '.' in timestamp:
        timestamp = timestamp.split('.')[0]
    
    try:
        from datetime import datetime
        dt = datetime.fromisoformat(timestamp)
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    except:
        return timestamp
