#!/bin/bash
# Run the ZK Health CLI

# Setup environment
export API_URL=${API_URL:-"http://localhost:8080"}

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed."
    exit 1
fi

# Install requirements if not already installed
if [ ! -f ".requirements_installed" ]; then
    echo "Installing requirements..."
    pip install -r requirements.txt
    touch .requirements_installed
fi

# Run the CLI tool
python3 zk_health_cli.py "$@"
