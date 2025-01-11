#!/bin/bash

# Run reset_db.sh to reset the database
echo "Resetting the database..."
./reset_db.sh

# Check if reset_db.sh ran successfully
if [ $? -ne 0 ]; then
  echo "Failed to reset the database. Exiting..."
  exit 1
fi

echo "Database has been reset successfully."

# Run the Go server
echo "Starting the Go server..."
go run ./cmd/server/main.go

# Check if Go server started successfully
if [ $? -ne 0 ]; then
  echo "Server is closed or encountered a error. Exiting..."
  exit 1
fi