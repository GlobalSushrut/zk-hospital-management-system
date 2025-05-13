#!/usr/bin/env python3
"""
ZK Health - Hospital Management System Frontend
This application serves as the frontend for the ZK Health Infrastructure, providing
a secure and privacy-focused interface for hospital management.
"""

import os
import uvicorn
from fastapi import FastAPI, Request, Depends, HTTPException, status
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from fastapi.middleware.cors import CORSMiddleware
from datetime import datetime, timedelta

# Import routers
from app.auth import router as auth_router
from app.dashboard import router as dashboard_router
from app.patients import router as patients_router
from app.consultations import router as consultations_router
from app.documents import router as documents_router
from app.treatments import router as treatments_router
from app.policies import router as policies_router
from app.oracle import router as oracle_router
from app.analytics import router as analytics_router
from app.admin import router as admin_router

# Import config and utilities
from utils.config import settings
from utils.auth import get_current_user

# Create FastAPI app
app = FastAPI(
    title="ZK Health - Hospital Management System",
    description="A secure, privacy-focused hospital management system using ZK-Proof technology",
    version="1.0.0"
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, you should restrict this
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Set up static files
app.mount("/static", StaticFiles(directory="static"), name="static")

# Set up templates
templates = Jinja2Templates(directory="templates")

# Include routers
app.include_router(auth_router.router, prefix="/auth", tags=["Authentication"])
app.include_router(dashboard_router.router, prefix="/dashboard", tags=["Dashboard"])
app.include_router(patients_router.router, prefix="/patients", tags=["Patient Management"])
app.include_router(consultations_router.router, prefix="/consultations", tags=["Consultations"])
app.include_router(documents_router.router, prefix="/documents", tags=["Document Management"])
app.include_router(treatments_router.router, prefix="/treatments", tags=["Treatment Plans"])
app.include_router(policies_router.router, prefix="/policies", tags=["Policy Management"])
app.include_router(oracle_router.router, prefix="/oracle", tags=["Oracle Agreements"])
app.include_router(analytics_router.router, prefix="/analytics", tags=["Analytics"])
app.include_router(admin_router.router, prefix="/admin", tags=["Administration"])

@app.get("/")
async def root(request: Request):
    """Root endpoint - redirects to login page"""
    return templates.TemplateResponse(
        "index.html", 
        {"request": request, "title": "ZK Health - Secure Hospital Management"}
    )

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "timestamp": datetime.now().isoformat(),
        "version": "1.0.0"
    }

if __name__ == "__main__":
    # Run the application
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG
    )
