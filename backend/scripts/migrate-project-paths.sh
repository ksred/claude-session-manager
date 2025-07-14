#!/bin/bash

# Script to run the project path migration
# This updates session project_path values using CWD data from messages

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Claude Session Manager - Project Path Migration${NC}"
echo "================================================"

# Change to backend directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Build the migration tool
echo -e "\n${YELLOW}Building migration tool...${NC}"
go build -o bin/migrate-project-paths cmd/migrate-project-paths/main.go

# Check if build was successful
if [ ! -f bin/migrate-project-paths ]; then
    echo -e "${RED}Failed to build migration tool${NC}"
    exit 1
fi

echo -e "${GREEN}Build successful!${NC}"

# Run the migration
echo -e "\n${YELLOW}Running migration...${NC}"

# First do a dry run
echo -e "\n${YELLOW}Performing dry run to show what will be changed...${NC}"
./bin/migrate-project-paths -dry-run "$@"

# Ask for confirmation
echo -e "\n${YELLOW}Do you want to proceed with the migration? (y/N)${NC}"
read -r response

if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo -e "\n${YELLOW}Applying migration...${NC}"
    ./bin/migrate-project-paths "$@"
    echo -e "\n${GREEN}Migration completed!${NC}"
else
    echo -e "\n${YELLOW}Migration cancelled.${NC}"
fi

# Clean up
rm -f bin/migrate-project-paths