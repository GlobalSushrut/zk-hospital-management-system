"""
Authentication router for ZK Health HMS
"""
from fastapi import APIRouter, Depends, HTTPException, status, Request, Form
from fastapi.security import OAuth2PasswordRequestForm
from fastapi.responses import RedirectResponse
from fastapi.templating import Jinja2Templates
from datetime import timedelta
from typing import Dict

from utils.auth import (
    get_password_hash, verify_password, create_access_token, 
    get_current_active_user
)
from utils.api_client import ZKIdentityClient, ZKGatewayClient
from utils.config import settings

router = APIRouter()
templates = Jinja2Templates(directory="templates")
identity_client = ZKIdentityClient()
gateway_client = ZKGatewayClient()

@router.get("/login")
async def login_page(request: Request):
    """Render login page"""
    return templates.TemplateResponse(
        "auth/login.html", 
        {"request": request, "title": "Login"}
    )

@router.post("/token")
async def login_for_access_token(form_data: OAuth2PasswordRequestForm = Depends()):
    """Handle login form submission and generate access token"""
    # Verify user identity with ZK Health Infrastructure
    user_response = await identity_client.verify_identity(form_data.username)
    
    if not user_response.get("success", False):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Incorrect username or password",
            headers={"WWW-Authenticate": "Bearer"},
        )
    
    user = user_response.get("data", {})
    stored_password_hash = user.get("password_hash", "")
    
    if not verify_password(form_data.password, stored_password_hash):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Incorrect username or password",
            headers={"WWW-Authenticate": "Bearer"},
        )
    
    # Generate access token using Gateway API
    token_data = {
        "user_id": user.get("id"),
        "role": user.get("role"),
        "country": user.get("country", settings.DEFAULT_COUNTRY)
    }
    
    # First, generate local token
    access_token_expires = timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    access_token = create_access_token(
        data={"sub": form_data.username, **token_data},
        expires_delta=access_token_expires
    )
    
    # Then, get token from ZK Gateway for API access
    gateway_token_response = await gateway_client.generate_token({
        "user_id": form_data.username,
        "role": user.get("role"),
        "country": user.get("country", settings.DEFAULT_COUNTRY)
    })
    
    gateway_token = gateway_token_response.get("token", "")
    
    return {
        "access_token": access_token,
        "gateway_token": gateway_token,
        "token_type": "bearer"
    }

@router.get("/register")
async def register_page(request: Request):
    """Render registration page"""
    return templates.TemplateResponse(
        "auth/register.html", 
        {"request": request, "title": "Register"}
    )

@router.post("/register")
async def register_user(
    request: Request,
    full_name: str = Form(...),
    email: str = Form(...),
    role: str = Form(...),
    country: str = Form(...),
    password: str = Form(...),
    confirm_password: str = Form(...)
):
    """Handle registration form submission"""
    if password != confirm_password:
        raise HTTPException(status_code=400, detail="Passwords do not match")
    
    # Hash the password
    hashed_password = get_password_hash(password)
    
    # Prepare data for ZK identity registration
    user_data = {
        "full_name": full_name,
        "email": email,
        "role": role,
        "country": country,
        "password_hash": hashed_password
    }
    
    # Register identity with ZK Health Infrastructure
    response = await identity_client.register_identity(user_data)
    
    if not response.get("success", False):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=response.get("error", "Registration failed")
        )
    
    # Redirect to login page
    return RedirectResponse(url="/auth/login", status_code=status.HTTP_303_SEE_OTHER)

@router.get("/logout")
async def logout():
    """Handle logout"""
    response = RedirectResponse(url="/auth/login", status_code=status.HTTP_303_SEE_OTHER)
    return response

@router.get("/profile")
async def profile(request: Request, current_user: Dict = Depends(get_current_active_user)):
    """User profile page"""
    return templates.TemplateResponse(
        "auth/profile.html", 
        {
            "request": request, 
            "title": "User Profile",
            "user": current_user
        }
    )
