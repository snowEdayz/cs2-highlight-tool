# PRD: Remove mistaken co-author trailer from commit 688ec20

## Goal
Remove `Co-authored-by: snow <snow@users.noreply.github.com>` from commit `688ec20768c74cbbd650827e4f93239f3474edca`.

## Constraints
- Do not change code content introduced by commit `688ec20`.
- Only rewrite commit message/history as needed.
- Keep branch functional and verifiable.

## Acceptance Criteria
1. Commit equivalent to `688ec20` no longer contains the `Co-authored-by: snow ...` trailer.
2. File tree and code diff relative to old history for that commit remains unchanged.
3. `git log --grep='Co-authored-by: snow <snow@users.noreply.github.com>'` returns no result on rewritten branch.
4. Branch is ready for force-push to update GitHub contributor attribution cache.
