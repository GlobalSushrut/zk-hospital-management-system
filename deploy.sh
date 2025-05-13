#!/bin/bash

# ZK-Proof-Based Decentralized Healthcare Infrastructure Deployment Script
# This script deploys the entire infrastructure to a Kubernetes cluster

set -e

echo "🚀 Deploying ZK-Proof-Based Decentralized Healthcare Infrastructure..."

# Check for kubectl
if ! command -v kubectl &> /dev/null; then
    echo "❌ Error: kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Check for connection to Kubernetes cluster
echo "🔍 Checking connection to Kubernetes cluster..."
if ! kubectl get nodes &> /dev/null; then
    echo "❌ Error: Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

# Create namespace if it doesn't exist
kubectl create namespace healthcare --dry-run=client -o yaml | kubectl apply -f -

# Build Docker image
echo "🔨 Building Docker image..."
docker build -t zkhealth:latest .

# Apply Kubernetes configurations
echo "📦 Deploying MongoDB..."
kubectl apply -f kubernetes/mongo-deployment.yaml -n healthcare

echo "📦 Deploying Cassandra..."
kubectl apply -f kubernetes/cassandra-deployment.yaml -n healthcare

echo "⏳ Waiting for databases to be ready..."
kubectl wait --for=condition=ready pod/mongodb-0 -n healthcare --timeout=300s
kubectl wait --for=condition=ready pod/cassandra-0 -n healthcare --timeout=300s

echo "📦 Deploying ZK Health API..."
kubectl apply -f kubernetes/zkhealth-deployment.yaml -n healthcare

echo "⏳ Waiting for ZK Health API to be ready..."
kubectl wait --for=condition=available deployment/zkhealth -n healthcare --timeout=300s

# Get the service URL
INGRESS_IP=$(kubectl get ingress zkhealth-ingress -n healthcare -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -z "$INGRESS_IP" ]; then
    echo "⚠️ Ingress IP not yet available. You may need to configure your DNS manually."
else
    echo "🔗 ZK Health API is available at: https://zkhealth.example.com"
    echo "  Add this to your /etc/hosts file for testing: $INGRESS_IP zkhealth.example.com"
fi

echo "✅ Deployment completed successfully!"
echo "🔐 The system is now ready for secure, privacy-preserving healthcare data management."
echo ""
echo "📊 Monitor the deployment with:"
echo "  kubectl get pods -n healthcare"
echo "  kubectl logs -f deployment/zkhealth -n healthcare"
echo ""
echo "🧪 Test the API with:"
echo "  curl -k https://zkhealth.example.com/health"
