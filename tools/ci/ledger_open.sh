#!/usr/bin/env bash
# Check out the ledger branch (append-only event files) into ./ledger-wt and point LEDGER_OUT at it.
set -e
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
if git fetch -q origin ledger:ledger 2>/dev/null; then git worktree add -q ledger-wt ledger
else git worktree add -q --orphan -b ledger ledger-wt; fi
echo "LEDGER_OUT=$PWD/ledger-wt" >> "${GITHUB_ENV:-/dev/null}"
