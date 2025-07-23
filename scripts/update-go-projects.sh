#!/bin/bash

# Script to add install targets to all Go project.json files

echo "🔧 Updating Go project configurations..."

# List of Go apps
GO_APPS=(
  "apps/subscription-processor"
  "apps/webhook-receiver"
  "apps/webhook-processor"
  "apps/dlq-processor"
  "libs/go"
)

# Function to add install target to project.json
add_install_target() {
  local project_path=$1
  local project_file="$project_path/project.json"
  
  if [ -f "$project_file" ]; then
    echo "Updating $project_file..."
    
    # Check if install target already exists
    if grep -q '"install"' "$project_file"; then
      echo "  ✓ Install target already exists"
    else
      # Add install target after "targets": {
      sed -i '' '/"targets": {/a\
    "install": {\
      "executor": "nx:run-commands",\
      "options": {\
        "command": "go mod tidy",\
        "cwd": "'"$project_path"'"\
      }\
    },
' "$project_file"
      echo "  ✓ Added install target"
    fi
  else
    echo "  ⚠️  $project_file not found"
  fi
}

# Update all Go projects
for app in "${GO_APPS[@]}"; do
  add_install_target "$app"
done

echo "✅ Done updating Go projects"