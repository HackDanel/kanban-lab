#!/bin/sh
# Daily PostgreSQL dump of each environment, kept for 14 days.
# Cron on the server: 30 3 * * * /opt/kanban/backup.sh
set -eu
BASE=${KANBAN_DIR:-/opt/kanban}
DIR=$BASE/backups
mkdir -p "$DIR"
for ENV in staging production; do
	docker compose -p "kanban-$ENV" -f "$BASE/docker-compose.yml" --env-file "$BASE/.env.$ENV" \
		exec -T postgres pg_dump -U kanban -Fc kanban > "$DIR/$ENV-$(date +%F).dump" || echo "backup of $ENV failed" >&2
done
find "$DIR" -name '*.dump' -mtime +14 -delete
