#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create databases for each service
    CREATE DATABASE controller_db;
    CREATE DATABASE textanalyzer_db;
    CREATE DATABASE scraper_db;
    CREATE DATABASE scheduler_db;

    -- Grant all privileges to the docutab user
    GRANT ALL PRIVILEGES ON DATABASE controller_db TO docutab;
    GRANT ALL PRIVILEGES ON DATABASE textanalyzer_db TO docutab;
    GRANT ALL PRIVILEGES ON DATABASE scraper_db TO docutab;
    GRANT ALL PRIVILEGES ON DATABASE scheduler_db TO docutab;
EOSQL

echo "All service databases created successfully"
