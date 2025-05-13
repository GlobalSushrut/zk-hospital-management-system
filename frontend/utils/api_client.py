"""
API Client utilities for interacting with ZK Health Infrastructure
"""
import json
import httpx
from typing import Dict, List, Any, Optional
from utils.config import settings

class ZKBaseClient:
    """Base client for ZK Health API interactions"""
    
    def __init__(self, base_url: str = None):
        self.base_url = base_url if base_url else settings.ZK_API_BASE_URL
        self.headers = {
            "Content-Type": "application/json",
            "Accept": "application/json"
        }
    
    async def _make_request(self, method: str, endpoint: str, data: Any = None, 
                           params: Dict = None, headers: Dict = None) -> Dict:
        """Make HTTP request to API"""
        url = f"{self.base_url}{endpoint}"
        request_headers = self.headers.copy()
        
        if headers:
            request_headers.update(headers)
        
        async with httpx.AsyncClient() as client:
            response = await client.request(
                method=method,
                url=url,
                json=data,
                params=params,
                headers=request_headers,
                timeout=30.0
            )
            
            if response.status_code >= 400:
                error_msg = f"API Error: {response.status_code} - {response.text}"
                print(error_msg)  # Log error
                return {"success": False, "error": error_msg}
            
            try:
                return response.json()
            except:
                return {"success": True, "data": response.text}


class ZKIdentityClient(ZKBaseClient):
    """Client for Identity API interactions"""
    
    def __init__(self):
        super().__init__(settings.IDENTITY_API)
    
    async def register_identity(self, user_data: Dict) -> Dict:
        """Register new identity"""
        return await self._make_request("POST", "/register", data=user_data)
    
    async def verify_identity(self, user_id: str) -> Dict:
        """Verify identity"""
        return await self._make_request("POST", "/verify", data={"user_id": user_id})
    
    async def get_identity(self, user_id: str) -> Dict:
        """Get identity details"""
        return await self._make_request("GET", f"/{user_id}")
    
    async def update_identity(self, user_id: str, update_data: Dict) -> Dict:
        """Update identity"""
        return await self._make_request("PUT", f"/{user_id}", data=update_data)
    
    async def generate_proof(self, user_id: str, proof_type: str) -> Dict:
        """Generate ZK proof for identity"""
        return await self._make_request(
            "POST", 
            "/proof/generate", 
            data={"user_id": user_id, "proof_type": proof_type}
        )


class ZKConsentClient(ZKBaseClient):
    """Client for Consent API interactions"""
    
    def __init__(self):
        super().__init__(settings.CONSENT_API)
    
    async def create_consent(self, consent_data: Dict) -> Dict:
        """Create consent agreement"""
        return await self._make_request("POST", "/create", data=consent_data)
    
    async def approve_consent(self, consent_id: str, user_id: str) -> Dict:
        """Approve consent"""
        return await self._make_request(
            "POST", 
            "/approve", 
            data={"consent_id": consent_id, "user_id": user_id}
        )
    
    async def verify_consent(self, consent_id: str) -> Dict:
        """Verify consent status"""
        return await self._make_request("GET", f"/verify/{consent_id}")
    
    async def revoke_consent(self, consent_id: str, user_id: str) -> Dict:
        """Revoke consent"""
        return await self._make_request(
            "POST", 
            "/revoke", 
            data={"consent_id": consent_id, "user_id": user_id}
        )
    
    async def list_user_consents(self, user_id: str) -> Dict:
        """List all consents for a user"""
        return await self._make_request("GET", f"/user/{user_id}")


class ZKDocumentClient(ZKBaseClient):
    """Client for Document API interactions"""
    
    def __init__(self):
        super().__init__(settings.DOCUMENT_API)
    
    async def upload_document(self, document_data: Dict, document_file: bytes) -> Dict:
        """Upload medical document"""
        headers = {"Content-Type": "multipart/form-data"}
        data = {
            "metadata": json.dumps(document_data),
            "file": document_file
        }
        return await self._make_request("POST", "/upload", data=data, headers=headers)
    
    async def verify_document(self, document_id: str) -> Dict:
        """Verify document authenticity"""
        return await self._make_request("GET", f"/verify/{document_id}")
    
    async def get_document(self, document_id: str, user_id: str) -> Dict:
        """Get document"""
        return await self._make_request(
            "POST", 
            f"/{document_id}", 
            data={"user_id": user_id}
        )
    
    async def search_documents(self, query: Dict) -> Dict:
        """Search documents"""
        return await self._make_request("POST", "/search", data=query)


class ZKTreatmentClient(ZKBaseClient):
    """Client for Treatment API interactions"""
    
    def __init__(self):
        super().__init__(settings.TREATMENT_API)
    
    async def create_treatment_vector(self, treatment_data: Dict) -> Dict:
        """Create treatment vector"""
        return await self._make_request("POST", "/vector/create", data=treatment_data)
    
    async def update_treatment_vector(self, vector_id: str, update_data: Dict) -> Dict:
        """Update treatment vector"""
        return await self._make_request("PUT", f"/vector/{vector_id}", data=update_data)
    
    async def get_treatment_vector(self, vector_id: str) -> Dict:
        """Get treatment vector"""
        return await self._make_request("GET", f"/vector/{vector_id}")
    
    async def analyze_treatment_vectors(self, analysis_params: Dict) -> Dict:
        """Analyze treatment vectors"""
        return await self._make_request("POST", "/analyze", data=analysis_params)


class ZKOracleClient(ZKBaseClient):
    """Client for Oracle API interactions"""
    
    def __init__(self):
        super().__init__(settings.ORACLE_API)
    
    async def create_agreement(self, agreement_data: Dict) -> Dict:
        """Create oracle agreement"""
        return await self._make_request("POST", "/agreement/create", data=agreement_data)
    
    async def validate_agreement(self, agreement_id: str, validation_data: Dict) -> Dict:
        """Validate oracle agreement"""
        return await self._make_request(
            "POST", 
            f"/agreement/{agreement_id}/validate", 
            data=validation_data
        )
    
    async def get_agreement(self, agreement_id: str) -> Dict:
        """Get oracle agreement details"""
        return await self._make_request("GET", f"/agreement/{agreement_id}")
    
    async def list_agreements(self, query_params: Dict = None) -> Dict:
        """List oracle agreements"""
        return await self._make_request("GET", "/agreements", params=query_params)


class ZKPolicyClient(ZKBaseClient):
    """Client for Policy API interactions"""
    
    def __init__(self):
        super().__init__(settings.POLICY_API)
    
    async def validate_action(self, validation_request: Dict) -> Dict:
        """Validate action against policy"""
        return await self._make_request("POST", "/validate", data=validation_request)
    
    async def get_allowed_actions(self, role: str, location: str) -> Dict:
        """Get allowed actions for role and location"""
        return await self._make_request(
            "GET", 
            "/actions", 
            params={"role": role, "location": location}
        )
    
    async def validate_policy_with_oracle(self, validation_request: Dict) -> Dict:
        """Validate policy with oracle integration"""
        return await self._make_request(
            "POST", 
            "/validate/oracle", 
            data=validation_request
        )


class ZKGatewayClient(ZKBaseClient):
    """Client for Gateway API interactions"""
    
    def __init__(self):
        super().__init__(settings.GATEWAY_API)
    
    async def generate_token(self, token_request: Dict) -> Dict:
        """Generate access token"""
        return await self._make_request("POST", "/token/generate", data=token_request)
    
    async def validate_token(self, token: str) -> Dict:
        """Validate access token"""
        return await self._make_request(
            "POST", 
            "/token/validate", 
            data={"token": token}
        )
    
    async def revoke_token(self, token: str) -> Dict:
        """Revoke access token"""
        return await self._make_request(
            "POST", 
            "/token/revoke", 
            data={"token": token}
        )
