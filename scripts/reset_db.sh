#!/bin/bash

# Database connection parameters
DB_HOST="localhost"
// DB_HOST="cyphera-db.cjwqc0yo6vzw.us-east-1.rds.amazonaws.com"
DB_PORT="5432"
DB_USER="postgres"
DB_NAME="cyphera"
DB_USER="apiuser"


# Path to init SQL script
INIT_SCRIPT="internal/db/init-scripts/01-init.sql"

# Prompt for password
echo -n "Enter database password: "
read -s DB_PASSWORD
echo

echo "Connecting to database and resetting data..."

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
