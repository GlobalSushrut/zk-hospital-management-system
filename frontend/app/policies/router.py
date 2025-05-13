"""
Policies router for ZK Health HMS
"""
from fastapi import APIRouter, Depends, Request, Form, HTTPException, status
from fastapi.templating import Jinja2Templates
from fastapi.responses import RedirectResponse, JSONResponse
from typing import Dict, List, Optional
import json
import uuid
from datetime import datetime

from utils.auth import get_current_active_user
from utils.api_client import ZKPolicyClient, ZKOracleClient

router = APIRouter()
templates = Jinja2Templates(directory="templates")

# Initialize API clients
policy_client = ZKPolicyClient()
oracle_client = ZKOracleClient()

@router.get("/")
async def policy_dashboard(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Policy dashboard view"""
    # Only users with admin or compliance officer roles can view policy dashboard
    if current_user.get("role") not in ["admin", "compliance_officer"]:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to access the policy dashboard"
        )
    
    # Get list of supported countries
    countries = [
        {"code": "IN", "name": "India", "flag_emoji": "🇮🇳"},
        {"code": "US", "name": "United States", "flag_emoji": "🇺🇸"},
        {"code": "CA", "name": "Canada", "flag_emoji": "🇨🇦"},
        {"code": "GB", "name": "United Kingdom", "flag_emoji": "🇬🇧"},
        {"code": "AU", "name": "Australia", "flag_emoji": "🇦🇺"}
    ]
    
    # Get list of defined roles
    roles = [
        {"id": "general_doctor", "name": "General Physician", "strength": 5},
        {"id": "specialist", "name": "Specialist", "strength": 8},
        {"id": "nurse", "name": "Nurse", "strength": 3},
        {"id": "admin", "name": "Administrator", "strength": 2},
        {"id": "researcher", "name": "Researcher", "strength": 4},
        {"id": "compliance_officer", "name": "Compliance Officer", "strength": 6}
    ]
    
    # Get list of actions
    actions = [
        {"id": "prescribe", "name": "Prescribe Medication", "min_strength": 5},
        {"id": "diagnose", "name": "Diagnose Patient", "min_strength": 5},
        {"id": "issue_certificate", "name": "Issue Medical Certificate", "min_strength": 8},
        {"id": "refer", "name": "Refer Patient", "min_strength": 5},
        {"id": "access_records", "name": "Access Medical Records", "min_strength": 3},
        {"id": "edit_records", "name": "Edit Medical Records", "min_strength": 5}
    ]
    
    # Get validator organizations
    validators = [
        {"id": "mci_validator", "name": "Medical Council of India", "country": "IN"},
        {"id": "health_canada", "name": "Health Canada", "country": "CA"},
        {"id": "us_hhs", "name": "US Department of Health & Human Services", "country": "US"},
        {"id": "nhs_validator", "name": "National Health Service UK", "country": "GB"},
        {"id": "australia_medical", "name": "Australian Medical Board", "country": "AU"}
    ]
    
    return templates.TemplateResponse(
        "policies/dashboard.html",
        {
            "request": request,
            "title": "Policy Management Dashboard",
            "user": current_user,
            "countries": countries,
            "roles": roles,
            "actions": actions,
            "validators": validators
        }
    )

@router.get("/validation-simulator")
async def validation_simulator(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Policy validation simulator"""
    # Get list of supported countries
    countries = [
        {"code": "IN", "name": "India"},
        {"code": "US", "name": "United States"},
        {"code": "CA", "name": "Canada"},
        {"code": "GB", "name": "United Kingdom"},
        {"code": "AU", "name": "Australia"}
    ]
    
    # Get list of defined roles
    roles = [
        {"id": "general_doctor", "name": "General Physician"},
        {"id": "specialist", "name": "Specialist"},
        {"id": "nurse", "name": "Nurse"},
        {"id": "admin", "name": "Administrator"},
        {"id": "researcher", "name": "Researcher"}
    ]
    
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
        "policies/simulator.html",
        {
            "request": request,
            "title": "Policy Validation Simulator",
            "user": current_user,
            "countries": countries,
            "roles": roles,
            "actions": actions
        }
    )

@router.post("/simulate")
async def simulate_validation(
    request: Request,
    role: str = Form(...),
    action: str = Form(...),
    country: str = Form(...),
    cross_jurisdiction: Optional[str] = Form(None),
    current_user: Dict = Depends(get_current_active_user)
):
    """Simulate policy validation"""
    # Prepare validation request
    validation_request = {
        "actor": {
            "id": f"actor-{uuid.uuid4().hex[:8]}",
            "role": role,
            "attributes": {
                "country": country
            }
        },
        "action": action,
        "location": country,
        "resource": {
            "id": f"resource-{uuid.uuid4().hex[:8]}",
            "type": "medical_record"
        },
        "timestamp": datetime.now().isoformat(),
        "client_address": request.client.host
    }
    
    # Add cross-jurisdiction if provided
    if cross_jurisdiction:
        validation_request["cross_jurisdiction"] = cross_jurisdiction
    
    # Validate against policy engine
    policy_response = await policy_client.validate_action(validation_request)
    
    # For demonstration purposes, integrate with Oracle Chain Validator
    if policy_response.get("allowed", False):
        oracle_validation_request = {
            "policy_request": validation_request,
            "agreement_id": f"oracle_agreement_{country}_{action}",
            "clause_ids": [f"clause_{country}_{action}_1", f"clause_{country}_{action}_2"]
        }
        
        oracle_response = await policy_client.validate_policy_with_oracle(oracle_validation_request)
        
        # Combine responses for display
        combined_response = {
            "policy_validation": policy_response,
            "oracle_validation": oracle_response
        }
    else:
        combined_response = {
            "policy_validation": policy_response,
            "oracle_validation": None
        }
    
    return JSONResponse(content=combined_response)

@router.get("/allowed-actions")
async def get_allowed_actions(
    request: Request,
    role: str,
    country: str,
    current_user: Dict = Depends(get_current_active_user)
):
    """Get allowed actions for a role in a country"""
    # Call policy client to get allowed actions
    response = await policy_client.get_allowed_actions(role, country)
    
    return response

@router.get("/cross-border")
async def cross_border_rules(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Cross-border policy rules view"""
    # Only users with admin or compliance officer roles can view cross-border rules
    if current_user.get("role") not in ["admin", "compliance_officer"]:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to access cross-border rules"
        )
    
    # Get list of supported countries
    countries = [
        {"code": "IN", "name": "India", "flag_emoji": "🇮🇳"},
        {"code": "US", "name": "United States", "flag_emoji": "🇺🇸"},
        {"code": "CA", "name": "Canada", "flag_emoji": "🇨🇦"},
        {"code": "GB", "name": "United Kingdom", "flag_emoji": "🇬🇧"},
        {"code": "AU", "name": "Australia", "flag_emoji": "🇦🇺"}
    ]
    
    # Example cross-border rules (in a real implementation, these would come from your ZK Policy Engine)
    cross_border_rules = [
        {
            "source": "US",
            "target": "CA",
            "actions": ["prescribe", "diagnose", "refer"],
            "roles": ["specialist"],
            "requirements": ["Medical license verification", "Cross-border authorization"],
            "validators": ["US HHS", "Health Canada"]
        },
        {
            "source": "CA",
            "target": "US",
            "actions": ["diagnose", "refer"],
            "roles": ["specialist", "general_doctor"],
            "requirements": ["Medical license verification", "State-specific authorization"],
            "validators": ["Health Canada", "US HHS"]
        },
        {
            "source": "IN",
            "target": "GB",
            "actions": ["diagnose", "refer"],
            "roles": ["specialist"],
            "requirements": ["MCI registration", "PLAB certification"],
            "validators": ["Medical Council of India", "NHS"]
        }
    ]
    
    return templates.TemplateResponse(
        "policies/cross_border.html",
        {
            "request": request,
            "title": "Cross-Border Policy Rules",
            "user": current_user,
            "countries": countries,
            "cross_border_rules": cross_border_rules
        }
    )

@router.get("/audit")
async def policy_audit_log(
    request: Request, 
    current_user: Dict = Depends(get_current_active_user)
):
    """Policy audit log view"""
    # Only users with admin or compliance officer roles can view audit logs
    if current_user.get("role") not in ["admin", "compliance_officer"]:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to access policy audit logs"
        )
    
    # In a real implementation, you'd fetch actual audit logs from your ZK Policy Engine
    # For demo, we'll return mock data
    audit_logs = [
        {
            "id": "log1",
            "timestamp": "2025-05-13T05:42:12",
            "actor_id": "dr_smith",
            "actor_role": "specialist",
            "action": "prescribe",
            "resource_id": "patient_record_1234",
            "location": "US",
            "allowed": True,
            "validator": "US Department of Health & Human Services"
        },
        {
            "id": "log2",
            "timestamp": "2025-05-13T04:58:37",
            "actor_id": "dr_patel",
            "actor_role": "general_doctor",
            "action": "issue_certificate",
            "resource_id": "patient_record_5678",
            "location": "IN",
            "allowed": False,
            "reason": "Insufficient role strength"
        },
        {
            "id": "log3",
            "timestamp": "2025-05-12T15:22:05",
            "actor_id": "dr_wilson",
            "actor_role": "specialist",
            "action": "diagnose",
            "resource_id": "patient_record_9012",
            "location": "CA",
            "cross_jurisdiction": "US",
            "allowed": True,
            "validator": "Health Canada"
        }
    ]
    
    return templates.TemplateResponse(
        "policies/audit.html",
        {
            "request": request,
            "title": "Policy Audit Logs",
            "user": current_user,
            "audit_logs": audit_logs
        }
    )
