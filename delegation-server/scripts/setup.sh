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

# Check if NPM_TOKEN is set
if [ -z "${NPM_TOKEN}" ]; then
  echo "WARNING: NPM_TOKEN environment variable is not set!"
  echo "You may not be able to access private packages."
else
  echo "Creating .npmrc file..."
  echo "//registry.npmjs.org/:_authToken=${NPM_TOKEN}" > .npmrc

  echo "Contents of .npmrc:"
  cat .npmrc
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