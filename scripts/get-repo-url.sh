#!/bin/bash
# Resolve the repository web URL for the commit links in the UI.
# Prefers the remote the current branch tracks (so fork clones link to the
# fork where the commits actually live), falling back to origin.

remote=$(git config branch."$(git branch --show-current)".remote 2>/dev/null)
if [ -z "$remote" ]; then
    remote=origin
fi

url=$(git remote get-url "$remote" 2>/dev/null)
if [ -z "$url" ]; then
    exit 0
fi

# Normalize to a browsable URL: git@github.com:owner/repo.git and
# ssh://git@github.com/owner/repo.git become https://github.com/owner/repo.
echo "$url" |
    sed -e 's#^git@github.com:#https://github.com/#' \
        -e 's#^ssh://git@github.com/#https://github.com/#' \
        -e 's#\.git$##'
