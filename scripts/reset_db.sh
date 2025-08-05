#!/bin/bash

# Database connection parameters
# DB_HOST="localhost"
# DB_HOST="cyphera-db-dev.cjwqc0yo6vzw.us-east-1.rds.amazonaws.com"
DB_PORT="5432"
DB_NAME="cyphera"

# DB_USER="apiuser"
# DB_PASSWORD="apipassword"

# Path to init SQL script
INIT_SCRIPT="libs/go/db/init-scripts/01-init.sql"

echo "Connecting to database, dropping all tables and custom types..."

# Connect to the database and perform operations
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF
-- Drop all tables in the public schema
DO \$\$ DECLARE
    r RECORD;
BEGIN
    -- Disable foreign key checks while dropping tables
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END \$\$;

-- Drop all custom types
DO \$\$ DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT typname FROM pg_type WHERE typnamespace = 'public'::regnamespace AND typtype = 'e') LOOP
        EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
    END LOOP;
END \$\$;

-- Drop all extensions
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
EOF

echo "Database reset complete. Initializing with new schema..."

# Run the initialization script
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $INIT_SCRIPT

echo "Database initialization complete!"

# Check if we're running locally (docker-compose environment)
if [ "$DB_HOST" = "localhost" ] || [ -z "$DB_HOST" ]; then
    echo "Restarting PostgreSQL to clear cached query plans..."
    
    # Check if docker-compose is available and postgres container is running
    if command -v docker-compose &> /dev/null; then
        if docker-compose ps | grep -q cyphera-db; then
            docker-compose restart postgres
            echo "PostgreSQL restarted. Waiting for it to be ready..."
            sleep 3
            
            # Wait for PostgreSQL to be ready
            until PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\q' 2>/dev/null; do
                echo "Waiting for PostgreSQL to be ready..."
                sleep 1
            done
            echo "PostgreSQL is ready!"
        else
            echo "PostgreSQL container not found. Skipping restart."
        fi
    else
        echo "docker-compose not found. Skipping PostgreSQL restart."
    fi
else
    echo "Remote database detected. Skipping PostgreSQL restart."
    echo "Note: You may need to restart your application to clear connection pool caches."
fi

echo "Database reset and initialization complete!" 
