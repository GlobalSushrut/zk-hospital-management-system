"""
Patients router for ZK Health HMS
"""
from fastapi import APIRouter, Depends, Request, Form, HTTPException, status, File, UploadFile
from fastapi.templating import Jinja2Templates
from fastapi.responses import RedirectResponse
from typing import Dict, List, Optional
import json
import uuid
from datetime import datetime

from utils.auth import get_current_active_user
from utils.api_client import (
    ZKIdentityClient, ZKConsentClient, ZKDocumentClient, 
    ZKTreatmentClient, ZKPolicyClient
)

router = APIRouter()
templates = Jinja2Templates(directory="templates")

# Initialize API clients
identity_client = ZKIdentityClient()
consent_client = ZKConsentClient()
document_client = ZKDocumentClient()
treatment_client = ZKTreatmentClient()
policy_client = ZKPolicyClient()

@router.get("/")
async def patients_list(
    request: Request, 
    search: Optional[str] = None,
    current_user: Dict = Depends(get_current_active_user)
):
    """List patients view"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_patients",
        "location": current_user.get("country"),
        "resource": {"type": "patient_list"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view patient records"
        )
    
    # In a real implementation, you'd query the database with search filters
    # For demo, we'll return mock data
    patients = [
        {
            "id": "pat101",
            "full_name": "John Doe",
            "gender": "Male",
            "age": 42,
            "contact": "+1-555-123-4567",
            "address": "123 Main St, Springfield",
            "registered_date": "2024-11-15",
            "status": "Active"
        },
        {
            "id": "pat102",
            "full_name": "Jane Smith",
            "gender": "Female",
            "age": 35,
            "contact": "+1-555-987-6543",
            "address": "456 Oak Ave, Riverdale",
            "registered_date": "2025-01-22",
            "status": "Active"
        },
        {
            "id": "pat103",
            "full_name": "Robert Johnson",
            "gender": "Male",
            "age": 58,
            "contact": "+1-555-246-8102",
            "address": "789 Pine Blvd, Meadowville",
            "registered_date": "2024-09-05",
            "status": "Inactive"
        }
    ]
    
    # Filter by search term if provided
    if search:
        patients = [p for p in patients if search.lower() in p["full_name"].lower()]
    
    return templates.TemplateResponse(
        "patients/list.html",
        {
            "request": request,
            "title": "Patient Records",
            "user": current_user,
            "patients": patients,
            "search_term": search or ""
        }
    )

@router.get("/register")
async def register_patient_form(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Patient registration form"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "register_patient",
        "location": current_user.get("country"),
        "resource": {"type": "patient_registration"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to register new patients"
        )
    
    return templates.TemplateResponse(
        "patients/register.html",
        {
            "request": request,
            "title": "Register New Patient",
            "user": current_user
        }
    )

@router.post("/register")
async def register_patient(
    request: Request,
    full_name: str = Form(...),
    date_of_birth: str = Form(...),
    gender: str = Form(...),
    contact: str = Form(...),
    email: Optional[str] = Form(None),
    address: str = Form(...),
    emergency_contact: str = Form(...),
    medical_history: str = Form(...),
    id_document: UploadFile = File(...),
    current_user: Dict = Depends(get_current_active_user)
):
    """Handle patient registration form submission"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "register_patient",
        "location": current_user.get("country"),
        "resource": {"type": "patient_registration"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to register new patients"
        )
    
    # Generate patient ID
    patient_id = f"PAT{uuid.uuid4().hex[:8].upper()}"
    
    # Create patient record
    patient_data = {
        "id": patient_id,
        "full_name": full_name,
        "date_of_birth": date_of_birth,
        "gender": gender,
        "contact": contact,
        "email": email,
        "address": address,
        "emergency_contact": emergency_contact,
        "medical_history": medical_history,
        "registered_by": current_user.get("id"),
        "registered_date": datetime.now().isoformat(),
        "status": "Active"
    }
    
    # In a real implementation, you'd store the patient data in database
    
    # Upload ID document with ZK Document API
    document_content = await id_document.read()
    document_metadata = {
        "document_type": "patient_id",
        "patient_id": patient_id,
        "uploaded_by": current_user.get("id"),
        "filename": id_document.filename,
        "description": f"Identity document for {full_name}"
    }
    
    document_response = await document_client.upload_document(document_metadata, document_content)
    
    if not document_response.get("success", False):
        # Log error but continue
        print(f"Error uploading document: {document_response.get('error')}")
    
    # Create initial consent records
    consent_data = {
        "patient_id": patient_id,
        "provider_id": current_user.get("id"),
        "scope": "medical_records",
        "purpose": "Patient registration and initial care",
        "valid_from": datetime.now().isoformat(),
        "valid_until": None,  # No expiration
        "data_use_policy": "Primary care and emergency treatment"
    }
    
    consent_response = await consent_client.create_consent(consent_data)
    
    if not consent_response.get("success", False):
        # Log error but continue
        print(f"Error creating consent: {consent_response.get('error')}")
    
    # Redirect to patient detail page
    return RedirectResponse(
        url=f"/patients/{patient_id}", 
        status_code=status.HTTP_303_SEE_OTHER
    )

@router.get("/{patient_id}")
async def patient_detail(
    request: Request,
    patient_id: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Patient detail view"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_patient_detail",
        "location": current_user.get("country"),
        "resource": {"type": "patient_record", "id": patient_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view this patient's details"
        )
    
    # In a real implementation, you'd fetch actual patient data from your database
    # For demo, we'll return mock data for the specified patient ID
    patient = {
        "id": patient_id,
        "full_name": "John Doe" if patient_id == "pat101" else "Jane Smith",
        "date_of_birth": "1982-05-15" if patient_id == "pat101" else "1990-03-22",
        "gender": "Male" if patient_id == "pat101" else "Female",
        "contact": "+1-555-123-4567" if patient_id == "pat101" else "+1-555-987-6543",
        "email": "john.doe@example.com" if patient_id == "pat101" else "jane.smith@example.com",
        "address": "123 Main St, Springfield" if patient_id == "pat101" else "456 Oak Ave, Riverdale",
        "emergency_contact": "+1-555-765-4321" if patient_id == "pat101" else "+1-555-321-6789",
        "medical_history": "Hypertension, Allergies (Peanuts)" if patient_id == "pat101" else "Diabetes Type 2, Asthma",
        "status": "Active"
    }
    
    # Get patient's medical records
    medical_records = [
        {
            "id": "mr101",
            "type": "Examination",
            "date": "2025-04-15",
            "doctor": "Dr. Sarah Wilson",
            "notes": "Regular check-up, blood pressure normal",
            "prescriptions": "None"
        },
        {
            "id": "mr102",
            "type": "Treatment",
            "date": "2025-03-10",
            "doctor": "Dr. James Brown",
            "notes": "Treated for seasonal allergies",
            "prescriptions": "Cetirizine 10mg daily for 7 days"
        }
    ]
    
    # Get patient's documents
    documents = [
        {
            "id": "doc101",
            "name": "Blood Test Results",
            "date": "2025-04-12",
            "type": "Laboratory",
            "uploaded_by": "Dr. Sarah Wilson"
        },
        {
            "id": "doc102",
            "name": "Chest X-Ray",
            "date": "2025-03-08",
            "type": "Radiology",
            "uploaded_by": "Dr. James Brown"
        }
    ]
    
    # Get patient's consents
    consents = [
        {
            "id": "con101",
            "scope": "Medical records",
            "provider": "Springfield General Hospital",
            "status": "Active",
            "date_granted": "2025-01-10"
        },
        {
            "id": "con102",
            "scope": "Telemedicine consultation",
            "provider": "TeleMed Services",
            "status": "Active",
            "date_granted": "2025-02-15"
        }
    ]
    
    # Get active treatments
    treatments = [
        {
            "id": "tr101",
            "condition": "Hypertension",
            "start_date": "2025-02-01",
            "doctor": "Dr. Sarah Wilson",
            "status": "Active",
            "next_appointment": "2025-06-01"
        }
    ]
    
    return templates.TemplateResponse(
        "patients/detail.html",
        {
            "request": request,
            "title": f"Patient: {patient['full_name']}",
            "user": current_user,
            "patient": patient,
            "medical_records": medical_records,
            "documents": documents,
            "consents": consents,
            "treatments": treatments
        }
    )

@router.get("/{patient_id}/edit")
async def edit_patient_form(
    request: Request,
    patient_id: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Edit patient form"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "edit_patient",
        "location": current_user.get("country"),
        "resource": {"type": "patient_record", "id": patient_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to edit this patient's details"
        )
    
    # In a real implementation, you'd fetch actual patient data from your database
    # For demo, we'll return mock data for the specified patient ID
    patient = {
        "id": patient_id,
        "full_name": "John Doe" if patient_id == "pat101" else "Jane Smith",
        "date_of_birth": "1982-05-15" if patient_id == "pat101" else "1990-03-22",
        "gender": "Male" if patient_id == "pat101" else "Female",
        "contact": "+1-555-123-4567" if patient_id == "pat101" else "+1-555-987-6543",
        "email": "john.doe@example.com" if patient_id == "pat101" else "jane.smith@example.com",
        "address": "123 Main St, Springfield" if patient_id == "pat101" else "456 Oak Ave, Riverdale",
        "emergency_contact": "+1-555-765-4321" if patient_id == "pat101" else "+1-555-321-6789",
        "medical_history": "Hypertension, Allergies (Peanuts)" if patient_id == "pat101" else "Diabetes Type 2, Asthma",
        "status": "Active"
    }
    
    return templates.TemplateResponse(
        "patients/edit.html",
        {
            "request": request,
            "title": f"Edit Patient: {patient['full_name']}",
            "user": current_user,
            "patient": patient
        }
    )

@router.post("/{patient_id}/edit")
async def update_patient(
    request: Request,
    patient_id: str,
    full_name: str = Form(...),
    contact: str = Form(...),
    email: Optional[str] = Form(None),
    address: str = Form(...),
    emergency_contact: str = Form(...),
    medical_history: str = Form(...),
    status: str = Form(...),
    current_user: Dict = Depends(get_current_active_user)
):
    """Handle patient update form submission"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "edit_patient",
        "location": current_user.get("country"),
        "resource": {"type": "patient_record", "id": patient_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to edit this patient's details"
        )
    
    # Update patient data
    patient_data = {
        "id": patient_id,
        "full_name": full_name,
        "contact": contact,
        "email": email,
        "address": address,
        "emergency_contact": emergency_contact,
        "medical_history": medical_history,
        "status": status,
        "last_updated_by": current_user.get("id"),
        "last_updated_date": datetime.now().isoformat()
    }
    
    # In a real implementation, you'd update the patient record in database
    
    # Create audit record of the change
    # This would be logged in a real implementation
    
    # Redirect to patient detail page
    return RedirectResponse(
        url=f"/patients/{patient_id}", 
        status_code=status.HTTP_303_SEE_OTHER
    )
