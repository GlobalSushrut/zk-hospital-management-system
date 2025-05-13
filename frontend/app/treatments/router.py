"""
Treatments router for ZK Health HMS
"""
from fastapi import APIRouter, Depends, Request, Form, HTTPException, status
from fastapi.templating import Jinja2Templates
from fastapi.responses import RedirectResponse
from typing import Dict, List, Optional
import json
import uuid
from datetime import datetime, timedelta

from utils.auth import get_current_active_user
from utils.api_client import (
    ZKTreatmentClient, ZKPolicyClient, ZKConsentClient, ZKOracleClient
)

router = APIRouter()
templates = Jinja2Templates(directory="templates")

# Initialize API clients
treatment_client = ZKTreatmentClient()
policy_client = ZKPolicyClient()
consent_client = ZKConsentClient()
oracle_client = ZKOracleClient()

@router.get("/")
async def treatment_list(
    request: Request, 
    status_filter: Optional[str] = None,
    current_user: Dict = Depends(get_current_active_user)
):
    """List treatments view"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_treatments",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_list"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view treatment records"
        )
    
    # In a real implementation, you'd query the database with status filters
    # For demo, we'll return mock data
    treatments = [
        {
            "id": "tr101",
            "patient_id": "pat101",
            "patient_name": "John Doe",
            "condition": "Hypertension",
            "start_date": "2025-02-01",
            "end_date": None,
            "doctor_id": "doc201",
            "doctor_name": "Dr. Sarah Wilson",
            "status": "Active",
            "next_appointment": "2025-06-01"
        },
        {
            "id": "tr102",
            "patient_id": "pat102",
            "patient_name": "Jane Smith",
            "condition": "Diabetes Type 2",
            "start_date": "2025-01-15",
            "end_date": None,
            "doctor_id": "doc202",
            "doctor_name": "Dr. James Brown",
            "status": "Active",
            "next_appointment": "2025-05-20"
        },
        {
            "id": "tr103",
            "patient_id": "pat103",
            "patient_name": "Robert Johnson",
            "condition": "Fractured Wrist",
            "start_date": "2024-12-05",
            "end_date": "2025-02-15",
            "doctor_id": "doc203",
            "doctor_name": "Dr. Emily Chen",
            "status": "Completed",
            "next_appointment": None
        }
    ]
    
    # Filter by status if provided
    if status_filter:
        treatments = [t for t in treatments if t["status"].lower() == status_filter.lower()]
    
    return templates.TemplateResponse(
        "treatments/list.html",
        {
            "request": request,
            "title": "Treatment Plans",
            "user": current_user,
            "treatments": treatments,
            "status_filter": status_filter or "All"
        }
    )

@router.get("/create")
async def create_treatment_form(
    request: Request,
    patient_id: Optional[str] = None,
    current_user: Dict = Depends(get_current_active_user)
):
    """Create treatment plan form"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "create_treatment",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_plan"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to create treatment plans"
        )
    
    # Get patient details if patient_id is provided
    patient = None
    if patient_id:
        # In a real implementation, you'd fetch actual patient data from your database
        patient = {
            "id": patient_id,
            "full_name": "John Doe" if patient_id == "pat101" else "Jane Smith",
        }
    
    # Get list of conditions (in a real app, this would come from a medical database)
    conditions = [
        "Hypertension", "Diabetes Type 1", "Diabetes Type 2", "Asthma",
        "Coronary Heart Disease", "Chronic Kidney Disease", "COPD",
        "Depression", "Anxiety", "Arthritis", "Osteoporosis",
        "Alzheimer's Disease", "Parkinson's Disease", "Cancer",
        "HIV/AIDS", "Hepatitis", "Stroke", "Tuberculosis",
        "Influenza", "Pneumonia", "Fractured Bone"
    ]
    
    return templates.TemplateResponse(
        "treatments/create.html",
        {
            "request": request,
            "title": "Create Treatment Plan",
            "user": current_user,
            "patient": patient,
            "conditions": conditions
        }
    )

@router.post("/create")
async def create_treatment(
    request: Request,
    patient_id: str = Form(...),
    condition: str = Form(...),
    description: str = Form(...),
    treatment_plan: str = Form(...),
    medications: str = Form(...),
    start_date: str = Form(...),
    estimated_end_date: Optional[str] = Form(None),
    next_appointment: Optional[str] = Form(None),
    notes: Optional[str] = Form(None),
    current_user: Dict = Depends(get_current_active_user)
):
    """Handle treatment plan creation form submission"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "create_treatment",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_plan", "patient_id": patient_id}
    }
    
    # Use policy-oracle integration for validation
    oracle_validation_request = {
        "policy_request": policy_request,
        "agreement_id": f"treatment_agreement_{current_user.get('country')}",
        "clause_ids": ["treatment_authorization", "patient_consent_verification"]
    }
    
    policy_response = await policy_client.validate_policy_with_oracle(oracle_validation_request)
    
    if not policy_response.get("policy_result", {}).get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=f"Policy restriction: {policy_response.get('policy_result', {}).get('reason', 'Not authorized')}"
        )
    
    if not policy_response.get("oracle_validated", False):
        invalid_clauses = policy_response.get("invalid_clauses", [])
        clause_details = ", ".join(invalid_clauses)
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=f"Oracle validation failed for clauses: {clause_details}"
        )
    
    # Verify consent exists
    consent_response = await consent_client.verify_consent(f"consent_{patient_id}_{current_user.get('id')}")
    
    if not consent_response.get("valid", False):
        # If no consent exists, we need to create a consent request
        consent_data = {
            "patient_id": patient_id,
            "provider_id": current_user.get("id"),
            "scope": "treatment_plan",
            "purpose": f"Treatment for {condition}",
            "valid_from": datetime.now().isoformat(),
            "valid_until": None,  # No expiration
            "data_use_policy": "Treatment and follow-up care"
        }
        
        await consent_client.create_consent(consent_data)
        
        # For demo purposes, we'll proceed as if consent was granted
        # In a real implementation, you'd wait for patient approval
    
    # Generate treatment vector ID
    treatment_id = f"TV{uuid.uuid4().hex[:8].upper()}"
    
    # Create treatment vector using ZK Treatment API
    treatment_data = {
        "id": treatment_id,
        "patient_id": patient_id,
        "provider_id": current_user.get("id"),
        "condition": condition,
        "description": description,
        "treatment_plan": treatment_plan,
        "medications": medications,
        "start_date": start_date,
        "estimated_end_date": estimated_end_date,
        "next_appointment": next_appointment,
        "notes": notes,
        "status": "Active",
        "country": current_user.get("country"),
        "zk_proof": f"proof_{uuid.uuid4().hex}"  # In real implementation, this would be generated
    }
    
    treatment_response = await treatment_client.create_treatment_vector(treatment_data)
    
    if not treatment_response.get("success", False):
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to create treatment vector: {treatment_response.get('error', 'Unknown error')}"
        )
    
    # Redirect to treatment detail page
    return RedirectResponse(
        url=f"/treatments/{treatment_id}", 
        status_code=status.HTTP_303_SEE_OTHER
    )

@router.get("/{treatment_id}")
async def treatment_detail(
    request: Request,
    treatment_id: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Treatment detail view"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_treatment_detail",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_vector", "id": treatment_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view this treatment plan"
        )
    
    # Get treatment details from ZK Treatment API
    # In a real implementation, you'd fetch actual treatment data
    # For demo, we'll return mock data for the specified treatment ID
    treatment = {
        "id": treatment_id,
        "patient_id": "pat101",
        "patient_name": "John Doe",
        "condition": "Hypertension",
        "description": "Essential (primary) hypertension with systolic BP > 140",
        "treatment_plan": "Lifestyle modifications and medication therapy",
        "medications": "Lisinopril 10mg once daily",
        "start_date": "2025-02-01",
        "estimated_end_date": None,
        "next_appointment": "2025-06-01",
        "notes": "Patient should monitor BP daily and maintain low-sodium diet",
        "status": "Active",
        "doctor_id": "doc201",
        "doctor_name": "Dr. Sarah Wilson",
        "created_date": "2025-02-01",
        "last_updated": "2025-04-15"
    }
    
    # Get treatment history/updates
    treatment_updates = [
        {
            "id": "upd101",
            "date": "2025-04-15",
            "provider": "Dr. Sarah Wilson",
            "notes": "Patient responding well to medication. BP averaged 130/85 over past month.",
            "changes": "No changes to medication regimen required."
        },
        {
            "id": "upd102",
            "date": "2025-03-10",
            "provider": "Dr. Sarah Wilson",
            "notes": "Follow-up after starting medication. Initial response positive.",
            "changes": "Continued current medication dosage."
        }
    ]
    
    return templates.TemplateResponse(
        "treatments/detail.html",
        {
            "request": request,
            "title": f"Treatment Plan: {treatment['condition']}",
            "user": current_user,
            "treatment": treatment,
            "updates": treatment_updates
        }
    )

@router.get("/{treatment_id}/update")
async def update_treatment_form(
    request: Request,
    treatment_id: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Update treatment form"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "update_treatment",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_vector", "id": treatment_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to update this treatment plan"
        )
    
    # Get treatment details from ZK Treatment API
    # In a real implementation, you'd fetch actual treatment data
    # For demo, we'll return mock data for the specified treatment ID
    treatment = {
        "id": treatment_id,
        "patient_id": "pat101",
        "patient_name": "John Doe",
        "condition": "Hypertension",
        "description": "Essential (primary) hypertension with systolic BP > 140",
        "treatment_plan": "Lifestyle modifications and medication therapy",
        "medications": "Lisinopril 10mg once daily",
        "start_date": "2025-02-01",
        "estimated_end_date": None,
        "next_appointment": "2025-06-01",
        "notes": "Patient should monitor BP daily and maintain low-sodium diet",
        "status": "Active"
    }
    
    return templates.TemplateResponse(
        "treatments/update.html",
        {
            "request": request,
            "title": f"Update Treatment Plan",
            "user": current_user,
            "treatment": treatment
        }
    )

@router.post("/{treatment_id}/update")
async def update_treatment(
    request: Request,
    treatment_id: str,
    update_notes: str = Form(...),
    medications: str = Form(...),
    next_appointment: Optional[str] = Form(None),
    status: str = Form(...),
    current_user: Dict = Depends(get_current_active_user)
):
    """Handle treatment update form submission"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "update_treatment",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_vector", "id": treatment_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to update this treatment plan"
        )
    
    # Update treatment data using ZK Treatment API
    update_data = {
        "id": treatment_id,
        "update_notes": update_notes,
        "medications": medications,
        "next_appointment": next_appointment,
        "status": status,
        "updated_by": current_user.get("id"),
        "updated_at": datetime.now().isoformat()
    }
    
    # In a real implementation, you'd call the actual API
    # treatment_response = await treatment_client.update_treatment_vector(treatment_id, update_data)
    
    # For demo purposes, we'll simulate a successful update
    treatment_response = {"success": True, "message": "Treatment updated successfully"}
    
    if not treatment_response.get("success", False):
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to update treatment: {treatment_response.get('error', 'Unknown error')}"
        )
    
    # Redirect to treatment detail page
    return RedirectResponse(
        url=f"/treatments/{treatment_id}", 
        status_code=status.HTTP_303_SEE_OTHER
    )

@router.get("/analytics")
async def treatment_analytics(
    request: Request,
    condition: Optional[str] = None,
    timeframe: Optional[str] = None,
    current_user: Dict = Depends(get_current_active_user)
):
    """Treatment analytics view"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_treatment_analytics",
        "location": current_user.get("country"),
        "resource": {"type": "treatment_analytics"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view treatment analytics"
        )
    
    # Set default timeframe if not provided
    if not timeframe:
        timeframe = "6m"  # 6 months
    
    # In a real implementation, you'd query the ZK Treatment API for analytics
    # For demo, we'll return mock analytics data
    
    # Success rate by condition
    success_rates = {
        "Hypertension": 87,
        "Diabetes Type 2": 82,
        "Asthma": 90,
        "Influenza": 95,
        "Fractured Bone": 98
    }
    
    # Average treatment duration by condition (in days)
    avg_durations = {
        "Hypertension": "Ongoing",
        "Diabetes Type 2": "Ongoing",
        "Asthma": "Ongoing",
        "Influenza": 14,
        "Fractured Bone": 45
    }
    
    # Treatment counts by status
    status_counts = {
        "Active": 48,
        "Completed": 72,
        "On Hold": 5,
        "Cancelled": 3
    }
    
    # Treatment counts by condition
    condition_counts = {
        "Hypertension": 25,
        "Diabetes Type 2": 18,
        "Asthma": 15,
        "Influenza": 42,
        "Fractured Bone": 28
    }
    
    return templates.TemplateResponse(
        "treatments/analytics.html",
        {
            "request": request,
            "title": "Treatment Analytics",
            "user": current_user,
            "success_rates": success_rates,
            "avg_durations": avg_durations,
            "status_counts": status_counts,
            "condition_counts": condition_counts,
            "selected_condition": condition,
            "selected_timeframe": timeframe
        }
    )
