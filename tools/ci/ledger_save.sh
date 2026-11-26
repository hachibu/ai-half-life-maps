#!/usr/bin/env bash
# Commit new ledger files and push, rebasing on concurrent writers (event files never collide).
set -e
cd ledger-wt
git add -A
git diff --cached --quiet && { echo "ledger: nothing new"; exit 0; }
git commit -qm "${1:-ledger update}"
for i in 1 2 3 4 5 6; do
  git push -q origin HEAD:ledger && { echo "ledger: saved"; exit 0; }
  sleep $((i * 2)); git pull -q --rebase origin ledger
done
echo "ledger: push failed"; exit 1
