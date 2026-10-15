#!/bin/sh
# Runs on the server: pulls the images for IMAGE_TAG and restarts the stack of
# the given environment. If the new version does not become healthy, the
# previous tag is started again. The directory defaults to /opt/kanban and can
# be changed with KANBAN_DIR.
# Usage: deploy.sh <staging|production> <image-tag>
set -eu
ENV=$1
TAG=$2
DIR=${KANBAN_DIR:-/opt/kanban}
PROJECT="kanban-$ENV"
ENV_FILE="$DIR/.env.$ENV"
STATE="$DIR/.deployed-$ENV"

cd "$DIR"
run() {
	tag=$1
	shift
	IMAGE_TAG=$tag docker compose -p "$PROJECT" -f "$DIR/docker-compose.yml" --env-file "$ENV_FILE" "$@"
}

PREVIOUS=$(cat "$STATE" 2>/dev/null || true)
echo "deploying $PROJECT: ${PREVIOUS:-none} -> $TAG"

# Registries time out now and then. Pull with retries; if a pull still fails,
# "up" fetches only the images that are really missing and a failure there
# leads to the rollback below.
for attempt in 1 2 3; do
	if run "$TAG" pull --quiet; then
		break
	fi
	echo "pull failed (attempt $attempt)" >&2
	sleep 10
done
if run "$TAG" up -d --pull missing --remove-orphans --wait --wait-timeout 120; then
	echo "$TAG" > "$STATE"
	docker image prune -f >/dev/null
	echo "deployed $TAG"
	exit 0
fi

echo "new version is unhealthy" >&2
run "$TAG" logs --tail 50 backend >&2 || true
if [ -n "$PREVIOUS" ]; then
	echo "rolling back to $PREVIOUS" >&2
	run "$PREVIOUS" up -d --remove-orphans --wait --wait-timeout 120
fi
exit 1
