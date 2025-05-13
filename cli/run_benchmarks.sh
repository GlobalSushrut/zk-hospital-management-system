#!/bin/bash

# Run Benchmarks for ZK Health Infrastructure
# This script starts the Go backend server and runs benchmarks against it

echo "==============================================="
echo "ZK Health Infrastructure Benchmark Runner"
echo "==============================================="

# Check if backend server is running
if ! nc -z localhost 8080 >/dev/null 2>&1; then
    echo "Starting the Go backend server in the background..."
    # Navigate to the root directory
    cd /home/umesh/Documents/telemedicine_tech
    
    # Start the Go server in the background
    go run cmd/server/main.go &
    GO_SERVER_PID=$!
    
    echo "Waiting for server to initialize (10 seconds)..."
    sleep 10
    
    echo "Go backend server started with PID: $GO_SERVER_PID"
else
    echo "Go backend server is already running"
fi

# Navigate to the cli directory
cd /home/umesh/Documents/telemedicine_tech/cli

# Run the benchmark suite
echo
echo "Running benchmarks against the actual Go modules..."
echo "==============================================="

# Ask which components to benchmark
echo "Which components would you like to benchmark?"
echo "1) Identity Management"
echo "2) Document Management"
echo "3) Gateway Services"
echo "4) Policy Engine"
echo "5) All Components"
read -p "Enter your choice (1-5): " choice

iterations=100
case $choice in
    1)
        echo "Running Identity Management benchmarks..."
        python benchmark_identity.py $iterations
        ;;
    2)
        echo "Running Document Management benchmarks..."
        python benchmark_document.py $iterations
        ;;
    3)
        echo "Running Gateway Services benchmarks..."
        python benchmark_gateway.py $iterations
        ;;
    4)
        echo "Running Policy Engine benchmarks..."
        python benchmark_policy.py $iterations
        ;;
    5)
        echo "Running all benchmarks..."
        python benchmark_identity.py $iterations
        python benchmark_document.py $iterations
        python benchmark_gateway.py $iterations
        python benchmark_policy.py $iterations
        ;;
    *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
esac

echo
echo "==============================================="
echo "Benchmark completed!"
echo "==============================================="

# Ask if user wants to stop the Go server
if [ -n "$GO_SERVER_PID" ]; then
    read -p "Do you want to stop the Go backend server? (y/n): " stop_server
    if [ "$stop_server" = "y" ]; then
        echo "Stopping Go backend server (PID: $GO_SERVER_PID)..."
        kill $GO_SERVER_PID
        echo "Server stopped."
    else
        echo "Server is still running in the background (PID: $GO_SERVER_PID)."
        echo "To stop it later, run: kill $GO_SERVER_PID"
    fi
fi

exit 0
