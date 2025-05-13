"""
Treatment Vector commands for ZK Health CLI
"""

import os
import json
import click
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.progress import track
from rich import box

from utils import make_api_request, handle_api_error

console = Console()

@click.group(name="treatment")
def treatment_group():
    """Manage treatment vectors and AI recommendations"""
    pass

@treatment_group.command(name="start")
@click.option('--patient-id', required=True, help='ID of the patient')
@click.option('--symptom', required=True, help='Primary symptom or condition')
@click.option('--doctor-id', required=True, help='ID of the doctor starting treatment')
@click.option('--zk-proof', required=True, help='ZK proof of the doctor')
@click.option('--notes', help='Initial treatment notes')
def start_treatment(patient_id, symptom, doctor_id, zk_proof, notes):
    """Start a new treatment vector"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'patient_id': patient_id,
            'symptom': symptom,
            'doctor_id': doctor_id,
            'zk_proof': zk_proof
        }
        
        if notes:
            payload['notes'] = notes
            
        response = make_api_request(f"{api_url}/api/treatment/start", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Treatment vector started successfully![/green]")
            console.print(f"Vector ID: {data.get('vector_id', 'N/A')}")
            
            recommended_path = data.get('recommended_path', [])
            if recommended_path:
                console.print(f"\n[bold]YAG AI Recommended Treatment Path:[/bold]")
                for i, step in enumerate(recommended_path, 1):
                    console.print(f"{i}. {step}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="update")
@click.option('--vector-id', required=True, help='ID of the treatment vector')
@click.option('--doctor-id', required=True, help='ID of the doctor updating treatment')
@click.option('--zk-proof', required=True, help='ZK proof of the doctor')
@click.option('--action', required=True, help='Action taken in this treatment step')
@click.option('--notes', help='Notes for this treatment step')
def update_treatment(vector_id, doctor_id, zk_proof, action, notes):
    """Update a treatment vector with a new action"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'vector_id': vector_id,
            'doctor_id': doctor_id,
            'zk_proof': zk_proof,
            'action': action
        }
        
        if notes:
            payload['notes'] = notes
            
        response = make_api_request(f"{api_url}/api/treatment/update", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Treatment vector updated successfully![/green]")
            
            misalignment = data.get('misalignment', None)
            if misalignment is not None:
                if misalignment:
                    console.print(f"[yellow]⚠ Warning: Misalignment detected with recommended path![/yellow]")
                    console.print(f"Misalignment score: {data.get('misalignment_score', 'N/A')}")
                    console.print(f"Recommendation: {data.get('recommendation', 'N/A')}")
                else:
                    console.print(f"[green]✓ Action aligns with YAG AI recommendations[/green]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="complete")
@click.option('--vector-id', required=True, help='ID of the treatment vector')
@click.option('--doctor-id', required=True, help='ID of the doctor completing treatment')
@click.option('--zk-proof', required=True, help='ZK proof of the doctor')
@click.option('--outcome', required=True, 
              type=click.Choice(['resolved', 'improved', 'unchanged', 'worsened', 'referred']),
              help='Outcome of the treatment')
@click.option('--notes', help='Final notes for this treatment')
def complete_treatment(vector_id, doctor_id, zk_proof, outcome, notes):
    """Complete a treatment vector and record the outcome"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'vector_id': vector_id,
            'doctor_id': doctor_id,
            'zk_proof': zk_proof,
            'outcome': outcome
        }
        
        if notes:
            payload['notes'] = notes
            
        response = make_api_request(f"{api_url}/api/treatment/complete", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Treatment vector completed successfully![/green]")
            console.print(f"Vector ID: {vector_id}")
            console.print(f"Outcome: {outcome}")
            
            if data.get('learning_updated', False):
                console.print(f"[blue]YAG AI has updated its learning model based on this outcome[/blue]")
            
            # Show overall stats
            console.print("\n[bold]Treatment Summary:[/bold]")
            console.print(f"Duration: {data.get('duration_days', 'N/A')} days")
            console.print(f"Steps: {data.get('total_steps', 'N/A')}")
            console.print(f"Adherence to AI recommendations: {data.get('adherence_percentage', 'N/A')}%")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="add-feedback")
@click.option('--vector-id', required=True, help='ID of the treatment vector')
@click.option('--party-id', required=True, help='ID of the party providing feedback')
@click.option('--role', required=True, 
              type=click.Choice(['doctor', 'patient', 'nurse', 'specialist']),
              help='Role of the party providing feedback')
@click.option('--zk-proof', required=True, help='ZK proof of the party')
@click.option('--rating', required=True, type=click.IntRange(1, 5), help='Rating (1-5)')
@click.option('--feedback', required=True, help='Feedback text')
def add_feedback(vector_id, party_id, role, zk_proof, rating, feedback):
    """Add feedback to a treatment vector"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        payload = {
            'vector_id': vector_id,
            'party_id': party_id,
            'role': role,
            'zk_proof': zk_proof,
            'rating': rating,
            'feedback': feedback
        }
            
        response = make_api_request(f"{api_url}/api/treatment/feedback", method="POST", json=payload)
        
        if response.ok:
            data = response.json()
            console.print(f"[green]Feedback added successfully![/green]")
            console.print(f"Feedback ID: {data.get('feedback_id', 'N/A')}")
            
            if data.get('learning_updated', False):
                console.print(f"[blue]YAG AI has updated its learning model based on this feedback[/blue]")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="get")
@click.option('--vector-id', required=True, help='ID of the treatment vector')
def get_treatment(vector_id):
    """Get details of a treatment vector"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        response = make_api_request(f"{api_url}/api/treatment/{vector_id}", method="GET")
        
        if response.ok:
            vector = response.json()
            
            status_color = {
                'active': 'yellow',
                'completed': 'green',
                'abandoned': 'red'
            }.get(vector.get('status', ''), 'white')
            
            console.print(Panel.fit(
                f"[bold]Vector ID:[/bold] {vector.get('vector_id', 'N/A')}\n"
                f"[bold]Patient ID:[/bold] {vector.get('patient_id', 'N/A')}\n"
                f"[bold]Primary Symptom:[/bold] {vector.get('symptom', 'N/A')}\n"
                f"[bold]Doctor ID:[/bold] {vector.get('doctor_id', 'N/A')}\n"
                f"[bold]Status:[/bold] [{status_color}]{vector.get('status', 'N/A')}[/{status_color}]\n"
                f"[bold]Started:[/bold] {vector.get('start_date', 'N/A')}\n"
                f"[bold]Last Updated:[/bold] {vector.get('last_updated', 'N/A')}\n"
                f"[bold]Steps Completed:[/bold] {vector.get('steps_completed', 'N/A')}",
                title="Treatment Vector", border_style="blue"
            ))
            
            # Show recommended path
            recommended_path = vector.get('recommended_path', [])
            if recommended_path:
                console.print("\n[bold]YAG AI Recommended Path:[/bold]")
                for i, step in enumerate(recommended_path, 1):
                    console.print(f"{i}. {step}")
            
            # Show actual path
            actual_path = vector.get('actual_path', [])
            if actual_path:
                console.print("\n[bold]Actual Treatment Path:[/bold]")
                
                step_table = Table(show_header=True, header_style="bold blue", box=box.ROUNDED)
                step_table.add_column("#")
                step_table.add_column("Action")
                step_table.add_column("Date")
                step_table.add_column("Alignment")
                step_table.add_column("Notes")
                
                for i, step in enumerate(actual_path, 1):
                    alignment = step.get('alignment', True)
                    alignment_text = "✓" if alignment else "⚠"
                    alignment_color = "green" if alignment else "yellow"
                    
                    step_table.add_row(
                        str(i),
                        step.get('action', 'N/A'),
                        step.get('date', 'N/A'),
                        f"[{alignment_color}]{alignment_text}[/{alignment_color}]",
                        step.get('notes', '')
                    )
                
                console.print(step_table)
            
            # Show outcome if completed
            if vector.get('status') == 'completed':
                outcome = vector.get('outcome', {})
                console.print("\n[bold]Treatment Outcome:[/bold]")
                console.print(f"Result: {outcome.get('result', 'N/A')}")
                console.print(f"Completed Date: {outcome.get('completion_date', 'N/A')}")
                console.print(f"Notes: {outcome.get('notes', 'N/A')}")
            
            # Show feedback
            feedback = vector.get('feedback', [])
            if feedback:
                console.print("\n[bold]Feedback:[/bold]")
                
                feedback_table = Table(show_header=True, header_style="bold blue")
                feedback_table.add_column("From")
                feedback_table.add_column("Role")
                feedback_table.add_column("Rating")
                feedback_table.add_column("Comment")
                feedback_table.add_column("Date")
                
                for fb in feedback:
                    rating = int(fb.get('rating', 3))
                    stars = "★" * rating + "☆" * (5 - rating)
                    rating_color = "green" if rating >= 4 else "yellow" if rating >= 3 else "red"
                    
                    feedback_table.add_row(
                        fb.get('party_id', 'N/A'),
                        fb.get('role', 'N/A'),
                        f"[{rating_color}]{stars}[/{rating_color}]",
                        fb.get('feedback', 'N/A'),
                        fb.get('date', 'N/A')
                    )
                
                console.print(feedback_table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="list")
@click.option('--patient-id', help='Filter by patient ID')
@click.option('--doctor-id', help='Filter by doctor ID')
@click.option('--status', type=click.Choice(['active', 'completed', 'abandoned']), 
              help='Filter by treatment status')
def list_treatments(patient_id, doctor_id, status):
    """List treatment vectors"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {}
        if patient_id:
            params['patient_id'] = patient_id
        if doctor_id:
            params['doctor_id'] = doctor_id
        if status:
            params['status'] = status
            
        response = make_api_request(f"{api_url}/api/treatment/list", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            vectors = data.get('vectors', [])
            
            if not vectors:
                console.print("[yellow]No treatment vectors found.[/yellow]")
                return
                
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Vector ID")
            table.add_column("Patient ID")
            table.add_column("Symptom")
            table.add_column("Doctor ID")
            table.add_column("Status")
            table.add_column("Started")
            table.add_column("Steps")
            
            for vector in vectors:
                status_color = {
                    'active': 'yellow',
                    'completed': 'green',
                    'abandoned': 'red'
                }.get(vector.get('status', ''), 'white')
                
                table.add_row(
                    vector.get('vector_id', 'N/A'),
                    vector.get('patient_id', 'N/A'),
                    vector.get('symptom', 'N/A'),
                    vector.get('doctor_id', 'N/A'),
                    f"[{status_color}]{vector.get('status', 'N/A')}[/{status_color}]",
                    vector.get('start_date', 'N/A'),
                    str(vector.get('steps_completed', 0))
                )
            
            console.print(table)
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="recommend")
@click.option('--symptom', required=True, help='Primary symptom or condition')
@click.option('--patient-age', type=int, help='Patient age for more accurate recommendations')
@click.option('--patient-gender', type=click.Choice(['male', 'female', 'other']), 
              help='Patient gender for more accurate recommendations')
@click.option('--conditions', help='Comma-separated list of pre-existing conditions')
def get_recommendation(symptom, patient_age, patient_gender, conditions):
    """Get YAG AI treatment recommendations for a symptom"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {
            'symptom': symptom
        }
        
        if patient_age:
            params['age'] = patient_age
        if patient_gender:
            params['gender'] = patient_gender
        if conditions:
            params['conditions'] = conditions
            
        response = make_api_request(f"{api_url}/api/treatment/recommend", method="GET", params=params)
        
        if response.ok:
            data = response.json()
            paths = data.get('recommended_paths', [])
            
            if not paths or len(paths) == 0:
                console.print(f"[yellow]No recommendations available for symptom: {symptom}[/yellow]")
                return
                
            console.print(f"[bold]YAG AI Recommendations for:[/bold] {symptom}")
            
            # Show primary recommendation
            primary = paths[0]
            console.print(Panel.fit(
                "\n".join([f"{i+1}. {step}" for i, step in enumerate(primary.get('steps', []))]),
                title="Primary Recommended Path", 
                subtitle=f"Confidence: {primary.get('confidence', 0)}%",
                border_style="green"
            ))
            
            # Show alternative paths if available
            if len(paths) > 1:
                console.print("\n[bold]Alternative Treatment Paths:[/bold]")
                
                for i, path in enumerate(paths[1:], 1):
                    console.print(f"\n[bold]Alternative {i}[/bold] (Confidence: {path.get('confidence', 0)}%)")
                    for j, step in enumerate(path.get('steps', []), 1):
                        console.print(f"{j}. {step}")
            
            # Show sources if available
            sources = data.get('sources', [])
            if sources:
                console.print("\n[bold]Sources and Evidence:[/bold]")
                for source in sources:
                    console.print(f"• {source}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")

@treatment_group.command(name="stats")
@click.option('--doctor-id', help='Filter stats by doctor ID')
@click.option('--symptom', help='Filter stats by symptom')
def treatment_stats(doctor_id, symptom):
    """Get statistics on treatment vectors and outcomes"""
    api_url = os.getenv('API_URL', 'http://localhost:8080')
    
    try:
        params = {}
        if doctor_id:
            params['doctor_id'] = doctor_id
        if symptom:
            params['symptom'] = symptom
            
        response = make_api_request(f"{api_url}/api/treatment/stats", method="GET", params=params)
        
        if response.ok:
            stats = response.json()
            
            title = "Treatment Statistics"
            if doctor_id:
                title += f" for Doctor {doctor_id}"
            if symptom:
                title += f" for {symptom}"
                
            console.print(Panel.fit(
                f"[bold]Total Treatments:[/bold] {stats.get('total_treatments', 0)}\n"
                f"[bold]Active Treatments:[/bold] {stats.get('active_treatments', 0)}\n"
                f"[bold]Completed Treatments:[/bold] {stats.get('completed_treatments', 0)}\n"
                f"[bold]Average Steps Per Treatment:[/bold] {stats.get('avg_steps', 0)}\n"
                f"[bold]Average Treatment Duration:[/bold] {stats.get('avg_duration_days', 0)} days\n"
                f"[bold]Average YAG AI Adherence:[/bold] {stats.get('avg_adherence', 0)}%\n",
                title=title, border_style="blue"
            ))
            
            # Show outcome distribution
            outcomes = stats.get('outcome_distribution', {})
            if outcomes:
                console.print("\n[bold]Treatment Outcomes:[/bold]")
                
                outcome_table = Table(show_header=True, header_style="bold blue")
                outcome_table.add_column("Outcome")
                outcome_table.add_column("Count")
                outcome_table.add_column("Percentage")
                
                for outcome, count in outcomes.items():
                    percentage = count / stats.get('completed_treatments', 1) * 100
                    outcome_table.add_row(
                        outcome.capitalize(),
                        str(count),
                        f"{percentage:.1f}%"
                    )
                
                console.print(outcome_table)
            
            # Show most common treatments
            common_treatments = stats.get('common_treatments', [])
            if common_treatments:
                console.print("\n[bold]Most Common Treatment Paths:[/bold]")
                
                for i, treatment in enumerate(common_treatments[:5], 1):
                    console.print(f"\n{i}. [bold]{treatment.get('name', 'N/A')}[/bold] (Used {treatment.get('count', 0)} times)")
                    steps = treatment.get('steps', [])
                    for j, step in enumerate(steps, 1):
                        console.print(f"   {j}. {step}")
        else:
            handle_api_error(response)
    except Exception as e:
        console.print(f"[red]Error: {str(e)}[/red]")
