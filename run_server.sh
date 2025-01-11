#!/bin/bash

# Run the Go server
echo "Starting the Go server..."
go run ./cmd/server/main.go

# Check if Go server started successfully
if [ $? -ne 0 ]; then
  echo "Server is closed or encountered a error. Exiting..."
  exit 1
fi