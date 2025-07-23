#!/bin/bash
# migrate-to-new-structure.sh - Migrate to new monorepo structure

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Cyphera Monorepo Structure Migration ===${NC}"
echo ""

# Get confirmation
echo -e "${YELLOW}This script will:${NC}"
echo "1. Backup current configuration files"
echo "2. Install new package.json and Makefile"
echo "3. Update all project.json files"
echo "4. Clean up deprecated scripts"
echo "5. Update .gitignore"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Migration cancelled${NC}"
    exit 1
fi

# Create backup directory
BACKUP_DIR="./backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo -e "\n${YELLOW}Step 1: Backing up current files...${NC}"

# Backup current files
cp package.json "$BACKUP_DIR/" 2>/dev/null || true
cp Makefile "$BACKUP_DIR/" 2>/dev/null || true
cp -r scripts "$BACKUP_DIR/" 2>/dev/null || true

echo -e "${GREEN}✓ Backup created at $BACKUP_DIR${NC}"

# Check if new files exist
if [ ! -f "package.json.new" ] || [ ! -f "Makefile.new" ]; then
    echo -e "${RED}✗ New configuration files not found. Please run the standardization process first.${NC}"
    exit 1
fi

echo -e "\n${YELLOW}Step 2: Installing new configuration...${NC}"

# Install new files
cp package.json.new package.json
cp Makefile.new Makefile

# Update project.json files if they exist
for app in api delegation-server web-app subscription-processor webhook-receiver webhook-processor dlq-processor; do
    if [ -f "apps/$app/project.json.new" ]; then
        cp "apps/$app/project.json.new" "apps/$app/project.json"
        echo -e "${GREEN}✓ Updated apps/$app/project.json${NC}"
    fi
done

echo -e "\n${YELLOW}Step 3: Updating scripts...${NC}"

# Make new scripts executable
chmod +x scripts/load-env.sh 2>/dev/null || true

# Clean up deprecated scripts
DEPRECATED_SCRIPTS=(
    "start-dev-all.sh"
    "run-integration-test.sh"
    "test-web3auth-integration.sh"
)

for script in "${DEPRECATED_SCRIPTS[@]}"; do
    if [ -f "scripts/$script" ]; then
        mv "scripts/$script" "$BACKUP_DIR/scripts/" 2>/dev/null || true
        echo -e "${YELLOW}  Moved deprecated script: $script${NC}"
    fi
done

echo -e "\n${YELLOW}Step 4: Updating .gitignore...${NC}"

# Add new patterns to .gitignore if not present
GITIGNORE_ADDITIONS=(
    "# Nx"
    ".nx/"
    "nx-cloud.env"
    ""
    "# Environment files"
    ".env.local"
    "apps/*/.env.local"
    ""
    "# Build outputs"
    "dist/"
    "tmp/"
    "bootstrap"
)

for pattern in "${GITIGNORE_ADDITIONS[@]}"; do
    if ! grep -q "^$pattern$" .gitignore 2>/dev/null; then
        echo "$pattern" >> .gitignore
    fi
done

echo -e "${GREEN}✓ Updated .gitignore${NC}"

echo -e "\n${YELLOW}Step 5: Installing dependencies...${NC}"

# Install npm dependencies
npm install

echo -e "\n${YELLOW}Step 6: Creating example environment files...${NC}"

# Create example .env.local files
for app in delegation-server web-app; do
    if [ -f "apps/$app/.env.example" ] && [ ! -f "apps/$app/.env.local" ]; then
        cp "apps/$app/.env.example" "apps/$app/.env.local"
        echo -e "${GREEN}✓ Created apps/$app/.env.local from example${NC}"
    fi
done

echo -e "\n${YELLOW}Step 7: Cleaning up...${NC}"

# Clean Nx cache
npx nx reset

# Remove old files
rm -f package.json.new Makefile.new
rm -f apps/*/project.json.new

echo -e "\n${GREEN}=== Migration Complete ===${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "1. Review the backup at $BACKUP_DIR"
echo "2. Test the new commands:"
echo "   - npm run dev:all    (start all services)"
echo "   - npm run test:all   (run all tests)"
echo "   - npm run build:all  (build everything)"
echo "3. Check COMMANDS.md for full command reference"
echo "4. Update your team about the new structure"
echo ""
echo -e "${YELLOW}Old commands are still available through make for compatibility${NC}"