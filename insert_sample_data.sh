#!/bin/bash

# Script to run psql with specific options
PSQL_USER="izumikyouka001"
PSQL_DB="postgres"
SQL_FILE="sample_data.sql"

# Check if the SQL file exists
if [ ! -f "$SQL_FILE" ]; then
    echo "Error: SQL file '$SQL_FILE' not found!"
    exit 1
fi

# Execute the psql command
echo "Running SQL script: $SQL_FILE..."
psql -U "$PSQL_USER" -d "$PSQL_DB" -f "$SQL_FILE"

# Check exit status
if [ $? -eq 0 ]; then
    echo "Sample data inserted successfully."
else
    echo "Error: Failed to execute the SQL script."
    exit 1
fi