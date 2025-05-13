"""
Configuration settings for the ZK Health Hospital Management System
"""
import os
from pydantic import BaseSettings
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

class Settings(BaseSettings):
    """Application settings"""
    # Application settings
    APP_NAME: str = "ZK Health HMS"
    DEBUG: bool = os.getenv("DEBUG", "False").lower() == "true"
    HOST: str = os.getenv("HOST", "127.0.0.1")
    PORT: int = int(os.getenv("PORT", "8000"))
    
    # Security settings
    SECRET_KEY: str = os.getenv("SECRET_KEY", "your-secret-key-for-jwt")
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 60
    
    # ZK Health API endpoints
    ZK_API_BASE_URL: str = os.getenv("ZK_API_BASE_URL", "http://localhost:8080")
    IDENTITY_API: str = f"{ZK_API_BASE_URL}/api/identity"
    CONSENT_API: str = f"{ZK_API_BASE_URL}/api/consent"
    DOCUMENT_API: str = f"{ZK_API_BASE_URL}/api/document"
    TREATMENT_API: str = f"{ZK_API_BASE_URL}/api/treatment"
    ORACLE_API: str = f"{ZK_API_BASE_URL}/api/oracle"
    POLICY_API: str = f"{ZK_API_BASE_URL}/api/policy"
    GATEWAY_API: str = f"{ZK_API_BASE_URL}/api/gateway"
    
    # MongoDB settings
    MONGODB_URL: str = os.getenv("MONGODB_URL", "mongodb://localhost:27017")
    MONGODB_DB: str = os.getenv("MONGODB_DB", "zk_health_hms")
    
    # Default country code for location-based policies
    DEFAULT_COUNTRY: str = os.getenv("DEFAULT_COUNTRY", "US")
    
    class Config:
        """Pydantic config"""
        env_file = ".env"

# Create settings instance
settings = Settings()
