#!/bin/sh
# Forced command for the CI deploy key (authorized_keys: command="...").
#
#   echo "$GITHUB_TOKEN" | ssh server deploy <staging|production> <40-hex sha>
#
# The key alone cannot change what runs: the deployment files are downloaded
# from the repository at the given commit with the job's GITHUB_TOKEN, read
# from stdin, which only a workflow of this repository can obtain. The same
# token logs in to GHCR for the pull, then the login is removed.
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
IFS= read -r TOKEN || true
[ -n "$TOKEN" ] || { echo "GitHub token expected on stdin" >&2; exit 2; }

fetch() {
	curl -fsSL -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.github.raw+json" \
		"https://api.github.com/repos/$REPO/contents/$1?ref=$SHA" -o "$DIR/$2.new" &&
		mv "$DIR/$2.new" "$DIR/$2"
}
mkdir -p "$DIR/grafana/provisioning/datasources" "$DIR/grafana/provisioning/dashboards" "$DIR/grafana/dashboards"
fetch deploy/compose/docker-compose.yml docker-compose.yml
fetch deploy/deploy.sh deploy.sh
fetch deploy/backup.sh backup.sh
fetch deploy/prometheus/prometheus.yml prometheus.yml
fetch deploy/grafana/provisioning/datasources/prometheus.yml grafana/provisioning/datasources/prometheus.yml
fetch deploy/grafana/provisioning/dashboards/kanban.yml grafana/provisioning/dashboards/kanban.yml
fetch deploy/grafana/dashboards/kanban.json grafana/dashboards/kanban.json

# Public images pull without a login; private ones need it, so a failed login is only a warning.
printf '%s' "$TOKEN" | docker login ghcr.io -u ci --password-stdin >/dev/null 2>&1 || echo "warning: GHCR login failed" >&2
status=0
KANBAN_DIR=$DIR sh "$DIR/deploy.sh" "$ENV" "$SHA" || status=$?
docker logout ghcr.io >/dev/null 2>&1 || true
exit $status
