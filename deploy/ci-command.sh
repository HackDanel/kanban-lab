#!/bin/sh
# Forced command for the CI deploy key (authorized_keys: command="...").
# The key can only update the deployment files from the repository at a given
# commit and deploy that commit; it never gets a shell.
#
#   ssh server deploy <staging|production> <40-hex commit sha>
set -eu
DIR=${KANBAN_DIR:-/opt/kanban}
REPO=${KANBAN_REPO:-HackDanel/kanban-lab}

# shellcheck disable=SC2086
set -- $SSH_ORIGINAL_COMMAND
[ "$#" -eq 3 ] && [ "$1" = deploy ] || { echo "usage: deploy <staging|production> <sha>" >&2; exit 2; }
ENV=$2
SHA=$3
case "$ENV" in staging|production) ;; *) echo "unknown environment" >&2; exit 2 ;; esac
echo "$SHA" | grep -Eq '^[0-9a-f]{40}$' || { echo "sha must be 40 hex characters" >&2; exit 2; }

# Deployment files of exactly that commit.
fetch() {
	curl -fsSL "https://raw.githubusercontent.com/$REPO/$SHA/$1" -o "$DIR/$2.new" && mv "$DIR/$2.new" "$DIR/$2"
}
mkdir -p "$DIR/grafana/provisioning/datasources" "$DIR/grafana/provisioning/dashboards" "$DIR/grafana/dashboards"
fetch deploy/compose/docker-compose.yml docker-compose.yml
fetch deploy/deploy.sh deploy.sh
fetch deploy/backup.sh backup.sh
fetch deploy/prometheus/prometheus.yml prometheus.yml
fetch deploy/grafana/provisioning/datasources/prometheus.yml grafana/provisioning/datasources/prometheus.yml
fetch deploy/grafana/provisioning/dashboards/kanban.yml grafana/provisioning/dashboards/kanban.yml
fetch deploy/grafana/dashboards/kanban.json grafana/dashboards/kanban.json

KANBAN_DIR=$DIR exec sh "$DIR/deploy.sh" "$ENV" "$SHA"
