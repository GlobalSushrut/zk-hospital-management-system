"""
Dashboard router for ZK Health HMS
"""
from fastapi import APIRouter, Depends, Request
from fastapi.templating import Jinja2Templates
from typing import Dict

from utils.auth import get_current_active_user
from utils.api_client import (
    ZKIdentityClient, ZKConsentClient, ZKDocumentClient,
    ZKTreatmentClient, ZKOracleClient, ZKPolicyClient
)

router = APIRouter()
templates = Jinja2Templates(directory="templates")

# Initialize API clients
identity_client = ZKIdentityClient()
consent_client = ZKConsentClient()
document_client = ZKDocumentClient()
treatment_client = ZKTreatmentClient()
oracle_client = ZKOracleClient()
policy_client = ZKPolicyClient()

@router.get("/")
async def dashboard(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Main dashboard view"""
    # Get counts and summary information
    role = current_user.get("role", "")
    country = current_user.get("country", "")
    
    # Get allowed actions for current role and location
    policy_response = await policy_client.get_allowed_actions(role, country)
    allowed_actions = policy_response.get("actions", [])
    
    # Get recent patient consultations (if doctor)
    recent_consultations = []
    if role in ["general_doctor", "specialist"]:
        # In a real implementation, you'd fetch real data here
        recent_consultations = [
            {"id": "cons1", "patient_name": "John Doe", "date": "2025-05-12", "status": "Completed"},
            {"id": "cons2", "patient_name": "Jane Smith", "date": "2025-05-13", "status": "Scheduled"},
        ]
    
    # Get recent documents
    recent_documents = [
        {"id": "doc1", "name": "Medical Report - John Doe", "date": "2025-05-10", "type": "Report"},
        {"id": "doc2", "name": "Prescription - Jane Smith", "date": "2025-05-11", "type": "Prescription"},
    ]
    
    # Get active treatments
    active_treatments = [
        {"id": "tr1", "patient_name": "John Doe", "condition": "Hypertension", "status": "Active"},
        {"id": "tr2", "patient_name": "Jane Smith", "condition": "Diabetes Type 2", "status": "Active"},
    ]
    
    # Pending consent requests
    pending_consents = [
        {"id": "con1", "description": "Medical data access - John Doe", "status": "Pending"},
        {"id": "con2", "description": "Treatment plan modification - Jane Smith", "status": "Pending"},
    ]
    
    return templates.TemplateResponse(
        "dashboard/index.html",
        {
            "request": request,
            "title": "Dashboard",
            "user": current_user,
            "allowed_actions": allowed_actions,
            "recent_consultations": recent_consultations,
            "recent_documents": recent_documents,
            "active_treatments": active_treatments,
            "pending_consents": pending_consents,
            "role": role,
            "country": country
        }
    )

@router.get("/stats")
async def dashboard_stats(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Dashboard statistics"""
    # In a real implementation, you'd fetch this from your backend
    stats = {
        "total_patients": 245,
        "active_consultations": 42,
        "pending_documents": 15,
        "treatment_success_rate": 87.5,
        "consent_approval_rate": 92.3,
    }
    
    return stats

@router.get("/activity")
async def recent_activity(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Recent activity feed"""
    # In a real implementation, you'd fetch actual activity from your backend
    activities = [
        {"type": "consultation", "description": "New consultation scheduled with John Doe", "time": "10 minutes ago"},
        {"type": "document", "description": "Medical report uploaded for Jane Smith", "time": "30 minutes ago"},
        {"type": "consent", "description": "Consent request approved by Jane Smith", "time": "1 hour ago"},
        {"type": "treatment", "description": "Treatment plan updated for John Doe", "time": "2 hours ago"},
    ]
    
    return activities
