#!/bin/bash

# ZK Health HMS Startup Script
echo "=========================================="
echo "ZK Health Hospital Management System Setup"
echo "=========================================="

# Check if .env file exists, if not create it from example
if [ ! -f ".env" ]; then
    echo "Creating .env file from example..."
    cp .env.example .env
    echo "Please update the .env file with your configuration if needed."
fi

# Check if running with Docker or locally
echo
echo "How would you like to run the application?"
echo "1) Docker (recommended, requires Docker and Docker Compose)"
echo "2) Local development server"
read -p "Enter your choice (1/2): " choice

case $choice in
    1)
        echo
        echo "Starting ZK Health HMS with Docker..."
        docker-compose up -d
        
        echo
        echo "Waiting for services to start..."
        sleep 5
        
        echo
        echo "=========================================="
        echo "ZK Health HMS is now running!"
        echo "----------------------------------------"
        echo "Frontend: http://localhost:8000"
        echo "API: http://localhost:8080"
        echo "----------------------------------------"
        echo "To view logs: docker-compose logs -f"
        echo "To stop: docker-compose down"
        echo "=========================================="
        ;;
        
    2)
        echo
        echo "Setting up local development environment..."
        
        # Check if virtual environment exists
        if [ ! -d "venv" ]; then
            echo "Creating virtual environment..."
            python3 -m venv venv
        fi
        
        # Activate virtual environment
        echo "Activating virtual environment..."
        source venv/bin/activate
        
        # Install dependencies
        echo "Installing dependencies..."
        pip install -r requirements.txt
        
        echo
        echo "Starting local development server..."
        python main.py
        ;;
        
    *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
esac
