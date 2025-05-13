"""
Oracle Agreement router for ZK Health HMS
"""
from fastapi import APIRouter, Depends, Request, Form, HTTPException, status, File, UploadFile
from fastapi.templating import Jinja2Templates
from fastapi.responses import RedirectResponse, JSONResponse
from typing import Dict, List, Optional
import json
import uuid
from datetime import datetime

from utils.auth import get_current_active_user
from utils.api_client import ZKOracleClient, ZKPolicyClient

router = APIRouter()
templates = Jinja2Templates(directory="templates")

# Initialize API clients
oracle_client = ZKOracleClient()
policy_client = ZKPolicyClient()

@router.get("/")
async def oracle_dashboard(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Oracle agreement dashboard"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_oracle_agreements",
        "location": current_user.get("country"),
        "resource": {"type": "oracle_agreements"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view oracle agreements"
        )
    
    # In a real implementation, you'd query the Oracle API for actual agreements
    # For demo, we'll return mock data
    agreements = [
        {
            "id": "ora101",
            "name": "Standard Medical Consultation Agreement",
            "type": "consent",
            "country": "US",
            "clauses_count": 5,
            "status": "Active",
            "created_date": "2025-01-15"
        },
        {
            "id": "ora102",
            "name": "Cross-Border Telemedicine Agreement",
            "type": "legal_compliance",
            "country": "IN",
            "cross_jurisdiction": "US",
            "clauses_count": 8,
            "status": "Active",
            "created_date": "2025-02-22"
        },
        {
            "id": "ora103",
            "name": "Medication Prescription Protocol",
            "type": "medical_protocol",
            "country": "CA",
            "clauses_count": 6,
            "status": "Active",
            "created_date": "2025-03-10"
        }
    ]
    
    return templates.TemplateResponse(
        "oracle/index.html",
        {
            "request": request,
            "title": "Oracle Agreements",
            "user": current_user,
            "agreements": agreements
        }
    )

@router.get("/create")
async def create_agreement_form(
    request: Request,
    current_user: Dict = Depends(get_current_active_user)
):
    """Create oracle agreement form"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "create_oracle_agreement",
        "location": current_user.get("country"),
        "resource": {"type": "oracle_agreement"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to create oracle agreements"
        )
    
    # Get list of supported countries
    countries = [
        {"code": "IN", "name": "India"},
        {"code": "US", "name": "United States"},
        {"code": "CA", "name": "Canada"},
        {"code": "GB", "name": "United Kingdom"},
        {"code": "AU", "name": "Australia"}
    ]
    
    # Get list of agreement types
    agreement_types = [
        {"id": "consent", "name": "Consent Agreement"},
        {"id": "legal_compliance", "name": "Legal Compliance Protocol"},
        {"id": "medical_protocol", "name": "Medical Protocol"},
        {"id": "data_sharing", "name": "Data Sharing Agreement"},
        {"id": "research", "name": "Research Protocol"}
    ]
    
    return templates.TemplateResponse(
        "oracle/create.html",
        {
            "request": request,
            "title": "Create Oracle Agreement",
            "user": current_user,
            "countries": countries,
            "agreement_types": agreement_types
        }
    )

@router.post("/create")
async def create_agreement(
    request: Request,
    name: str = Form(...),
    description: str = Form(...),
    agreement_type: str = Form(...),
    country: str = Form(...),
    cross_jurisdiction: Optional[str] = Form(None),
    clauses: str = Form(...),
    current_user: Dict = Depends(get_current_active_user)
):
    """Handle agreement creation form submission"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "create_oracle_agreement",
        "location": current_user.get("country"),
        "resource": {"type": "oracle_agreement"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to create oracle agreements"
        )
    
    # Parse clauses
    try:
        clauses_list = json.loads(clauses)
    except json.JSONDecodeError:
        # If JSON parsing fails, try to split by newlines and create clauses
        clauses_list = [
            {"id": f"clause_{i+1}", "text": clause.strip()}
            for i, clause in enumerate(clauses.strip().split("\n"))
            if clause.strip()
        ]
    
    # Generate agreement ID
    agreement_id = f"ORA{uuid.uuid4().hex[:8].upper()}"
    
    # Create agreement data
    agreement_data = {
        "id": agreement_id,
        "name": name,
        "description": description,
        "type": agreement_type,
        "country": country,
        "cross_jurisdiction": cross_jurisdiction,
        "clauses": clauses_list,
        "created_by": current_user.get("id"),
        "created_date": datetime.now().isoformat(),
        "status": "Active"
    }
    
    # In a real implementation, you'd call the Oracle API to create the agreement
    # oracle_response = await oracle_client.create_agreement(agreement_data)
    
    # For demo purposes, we'll simulate a successful creation
    oracle_response = {"success": True, "id": agreement_id, "message": "Agreement created successfully"}
    
    if not oracle_response.get("success", False):
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to create agreement: {oracle_response.get('error', 'Unknown error')}"
        )
    
    # Redirect to agreement detail page
    return RedirectResponse(
        url=f"/oracle/{agreement_id}", 
        status_code=status.HTTP_303_SEE_OTHER
    )

@router.get("/{agreement_id}")
async def agreement_detail(
    request: Request,
    agreement_id: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Agreement detail view"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_oracle_agreement",
        "location": current_user.get("country"),
        "resource": {"type": "oracle_agreement", "id": agreement_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view this oracle agreement"
        )
    
    # Get agreement details from Oracle API
    # In a real implementation, you'd fetch actual agreement data
    # For demo, we'll return mock data for the specified agreement ID
    
    # Mock agreement data (would come from API in real implementation)
    agreement = None
    if agreement_id == "ora101":
        agreement = {
            "id": "ora101",
            "name": "Standard Medical Consultation Agreement",
            "description": "Standard agreement for medical consultations covering consent, privacy, and professional responsibilities.",
            "type": "consent",
            "country": "US",
            "created_by": "admin",
            "created_date": "2025-01-15",
            "status": "Active",
            "clauses": [
                {"id": "clause1", "text": "Patient provides informed consent for medical consultation."},
                {"id": "clause2", "text": "Doctor confirms identity and credentials are valid."},
                {"id": "clause3", "text": "Patient data will be handled according to HIPAA regulations."},
                {"id": "clause4", "text": "Consultation will be securely recorded with patient consent."},
                {"id": "clause5", "text": "Follow-up care instructions will be provided."}
            ]
        }
    elif agreement_id == "ora102":
        agreement = {
            "id": "ora102",
            "name": "Cross-Border Telemedicine Agreement",
            "description": "Agreement for telemedicine consultations across international borders.",
            "type": "legal_compliance",
            "country": "IN",
            "cross_jurisdiction": "US",
            "created_by": "admin",
            "created_date": "2025-02-22",
            "status": "Active",
            "clauses": [
                {"id": "clause1", "text": "Doctor confirms licensure is valid in both jurisdictions."},
                {"id": "clause2", "text": "Patient acknowledges cross-border nature of consultation."},
                {"id": "clause3", "text": "Medical advice follows stricter of the two jurisdiction's standards."},
                {"id": "clause4", "text": "Prescription medications must be legal in patient's jurisdiction."},
                {"id": "clause5", "text": "Emergency referral to local resources will be provided if needed."},
                {"id": "clause6", "text": "Data sovereignty requirements for both countries will be respected."},
                {"id": "clause7", "text": "Dispute resolution will follow international telemedicine standards."},
                {"id": "clause8", "text": "Billing and payments comply with both countries' regulations."}
            ]
        }
    elif agreement_id == "ora103":
        agreement = {
            "id": "ora103",
            "name": "Medication Prescription Protocol",
            "description": "Protocol for prescription medications including verification and monitoring procedures.",
            "type": "medical_protocol",
            "country": "CA",
            "created_by": "admin",
            "created_date": "2025-03-10",
            "status": "Active",
            "clauses": [
                {"id": "clause1", "text": "Prescriber must verify patient identity before prescribing."},
                {"id": "clause2", "text": "Patient medical history must be reviewed for contraindications."},
                {"id": "clause3", "text": "Prescription must comply with provincial formulary guidelines."},
                {"id": "clause4", "text": "Controlled substances require additional verification steps."},
                {"id": "clause5", "text": "Monitoring plan must be established for ongoing medications."},
                {"id": "clause6", "text": "Patient must be informed of potential side effects."}
            ]
        }
    else:
        # Generate a mock agreement if ID doesn't match known examples
        agreement = {
            "id": agreement_id,
            "name": f"Example Agreement {agreement_id}",
            "description": "Generic oracle agreement for demonstration purposes.",
            "type": "consent",
            "country": current_user.get("country"),
            "created_by": "admin",
            "created_date": "2025-04-01",
            "status": "Active",
            "clauses": [
                {"id": "clause1", "text": "Example clause 1 for this agreement."},
                {"id": "clause2", "text": "Example clause 2 for this agreement."},
                {"id": "clause3", "text": "Example clause 3 for this agreement."}
            ]
        }
    
    # Get validation history
    validation_history = [
        {
            "id": "val101",
            "timestamp": "2025-05-12T14:30:22",
            "actor_id": "doc201",
            "actor_name": "Dr. Sarah Wilson",
            "action": "prescribe",
            "status": "Valid",
            "clauses_validated": ["clause1", "clause2", "clause3", "clause5"]
        },
        {
            "id": "val102",
            "timestamp": "2025-05-10T09:15:48",
            "actor_id": "doc202",
            "actor_name": "Dr. James Brown",
            "action": "diagnose",
            "status": "Valid",
            "clauses_validated": ["clause1", "clause2", "clause4"]
        },
        {
            "id": "val103",
            "timestamp": "2025-05-05T16:22:10",
            "actor_id": "doc203",
            "actor_name": "Dr. Emily Chen",
            "action": "issue_certificate",
            "status": "Invalid",
            "clauses_validated": ["clause1", "clause2"],
            "clauses_failed": ["clause3"]
        }
    ]
    
    return templates.TemplateResponse(
        "oracle/detail.html",
        {
            "request": request,
            "title": f"Oracle Agreement: {agreement['name']}",
            "user": current_user,
            "agreement": agreement,
            "validation_history": validation_history
        }
    )

@router.get("/validate/{agreement_id}")
async def validate_agreement_form(
    request: Request,
    agreement_id: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Validate oracle agreement form"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "validate_oracle_agreement",
        "location": current_user.get("country"),
        "resource": {"type": "oracle_agreement", "id": agreement_id}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to validate this oracle agreement"
        )
    
    # Get agreement details
    # In a real implementation, you'd fetch actual agreement data
    # For demo, we'll return mock data for the specified agreement ID
    agreement = {
        "id": agreement_id,
        "name": f"Example Agreement {agreement_id}",
        "country": current_user.get("country")
    }
    
    # Get list of actions
    actions = [
        {"id": "prescribe", "name": "Prescribe Medication"},
        {"id": "diagnose", "name": "Diagnose Patient"},
        {"id": "issue_certificate", "name": "Issue Medical Certificate"},
        {"id": "refer", "name": "Refer Patient"},
        {"id": "access_records", "name": "Access Medical Records"},
        {"id": "edit_records", "name": "Edit Medical Records"}
    ]
    
    return templates.TemplateResponse(
        "oracle/validate.html",
        {
            "request": request,
            "title": "Validate Oracle Agreement",
            "user": current_user,
            "agreement": agreement,
            "actions": actions
        }
    )

@router.post("/validate/{agreement_id}")
async def validate_agreement(
    request: Request,
    agreement_id: str,
    action: str = Form(...),
    resource_id: str = Form(...),
    resource_type: str = Form(...),
    current_user: Dict = Depends(get_current_active_user)
):
    """Handle agreement validation form submission"""
    # Prepare validation data
    validation_data = {
        "agreement_id": agreement_id,
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {
                "country": current_user.get("country")
            }
        },
        "action": action,
        "resource": {
            "id": resource_id,
            "type": resource_type
        },
        "location": current_user.get("country"),
        "timestamp": datetime.now().isoformat()
    }
    
    # In a real implementation, you'd call the Oracle API to validate the agreement
    # response = await oracle_client.validate_agreement(agreement_id, validation_data)
    
    # For demo purposes, we'll simulate a response
    response = {
        "agreement_id": agreement_id,
        "validation_id": f"val_{uuid.uuid4().hex[:8]}",
        "timestamp": datetime.now().isoformat(),
        "valid": True,
        "validated_clauses": ["clause1", "clause2", "clause3"],
        "failed_clauses": [],
        "validation_details": {
            "actor_validated": True,
            "action_allowed": True,
            "resource_accessible": True
        }
    }
    
    return JSONResponse(content=response)

@router.get("/templates")
async def agreement_templates(
    request: Request,
    country: Optional[str] = None,
    type: Optional[str] = None,
    current_user: Dict = Depends(get_current_active_user)
):
    """Oracle agreement templates"""
    # Verify policy permission
    policy_request = {
        "actor": {
            "id": current_user.get("id"),
            "role": current_user.get("role"),
            "attributes": {"country": current_user.get("country")}
        },
        "action": "view_oracle_templates",
        "location": current_user.get("country"),
        "resource": {"type": "oracle_templates"}
    }
    
    policy_response = await policy_client.validate_action(policy_request)
    
    if not policy_response.get("allowed", False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Policy restriction: You are not authorized to view oracle templates"
        )
    
    # Get list of supported countries
    countries = [
        {"code": "IN", "name": "India"},
        {"code": "US", "name": "United States"},
        {"code": "CA", "name": "Canada"},
        {"code": "GB", "name": "United Kingdom"},
        {"code": "AU", "name": "Australia"}
    ]
    
    # Get list of template types
    template_types = [
        {"id": "consent", "name": "Consent Agreement"},
        {"id": "legal_compliance", "name": "Legal Compliance Protocol"},
        {"id": "medical_protocol", "name": "Medical Protocol"},
        {"id": "data_sharing", "name": "Data Sharing Agreement"},
        {"id": "research", "name": "Research Protocol"}
    ]
    
    # In a real implementation, you'd query the Oracle API for templates with filters
    # For demo, we'll return mock data
    templates = [
        {
            "id": "tpl101",
            "name": "Standard Medical Consultation",
            "description": "Template for standard medical consultations.",
            "type": "consent",
            "country": "US",
            "clauses_count": 5,
            "popularity": "High"
        },
        {
            "id": "tpl102",
            "name": "Cross-Border Telemedicine",
            "description": "Template for telemedicine across international borders.",
            "type": "legal_compliance",
            "country": "IN",
            "cross_jurisdiction": "US",
            "clauses_count": 8,
            "popularity": "Medium"
        },
        {
            "id": "tpl103",
            "name": "Medication Prescription Protocol",
            "description": "Standard protocol for medication prescriptions.",
            "type": "medical_protocol",
            "country": "CA",
            "clauses_count": 6,
            "popularity": "High"
        },
        {
            "id": "tpl104",
            "name": "Research Data Sharing",
            "description": "Template for sharing anonymized medical data for research.",
            "type": "data_sharing",
            "country": "GB",
            "clauses_count": 7,
            "popularity": "Medium"
        },
        {
            "id": "tpl105",
            "name": "Clinical Trial Protocol",
            "description": "Template for clinical trial participation agreement.",
            "type": "research",
            "country": "US",
            "clauses_count": 10,
            "popularity": "Low"
        }
    ]
    
    # Filter by country if provided
    if country:
        templates = [t for t in templates if t["country"] == country]
    
    # Filter by type if provided
    if type:
        templates = [t for t in templates if t["type"] == type]
    
    return templates
