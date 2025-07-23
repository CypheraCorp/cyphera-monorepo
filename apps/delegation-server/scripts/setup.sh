#!/bin/bash

set -e

# Change to the project root directory if running from scripts folder
if [[ "$(basename "$(pwd)")" == "scripts" ]]; then
  cd ..
fi

echo "Installing Node.js dependencies for the delegation server..."

# Check if the clean option is provided
if [ "$1" == "--clean" ]; then
  echo "Cleaning up existing node_modules directory..."
  rm -rf node_modules package-lock.json dist
fi

# Install dependencies without triggering the npm install script
echo "Installing dependencies with --legacy-peer-deps and ignoring scripts..."
npm install --legacy-peer-deps --ignore-scripts

# Build the project
echo "Building the project..."
npm run build

echo "Installation complete! You can now run the server with:"
echo "npm start       # For production"
echo "npm run dev     # For development"