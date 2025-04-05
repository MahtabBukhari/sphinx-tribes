#!/bin/bash

cd /workspaces

git clone https://github.com/stakwork/sphinx-tribes-frontend

cd sphinx-tribes

DB=postgres://postgres:postgres@localhost:5432/postgres

# Wait for PostgreSQL to become available
until psql $DB -c '\q'
do
  echo "Waiting for PostgreSQL to become available..."
  sleep 5
done

# Wait for backend to be ready
until [ "$(curl -s -m 1 http://localhost:5002/tribes 2>/dev/null)" = "[]" ]
do
  echo "Waiting for backend to become ready..."
  sleep 5
done

echo "Inserting dummy data...."

psql $DB -f docker/dummy-data/people.sql
psql $DB -f docker/dummy-data/workspaces.sql
psql $DB -f docker/dummy-data/paid-bounties.sql

./.devcontainer/ports.sh
