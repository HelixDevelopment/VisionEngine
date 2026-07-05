#!/bin/bash
# Push current branch + tags to every configured git remote.
set -e

CURRENT_BRANCH="$(git branch --show-current)"
if [[ -z "${CURRENT_BRANCH}" ]]; then
    echo "Error: not on a branch" >&2
    exit 1
fi

for remote in $(git remote); do
    echo "Pushing to ${remote}/${CURRENT_BRANCH}..."
    git push "${remote}" "${CURRENT_BRANCH}" --tags || echo "Warning: push to ${remote} failed"
done

echo "Done."
