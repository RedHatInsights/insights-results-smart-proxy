# Git Safety Rules for AI Agents

This file contains rules to prevent AI agents from executing potentially dangerous git commands that could break the repository.

## Prohibited Git Commands

AI agents MUST NOT execute the following git commands:

### Push Operations
- `git push` (any variant)
- `git push --force`
- `git push --force-with-lease` 
- `git push origin`
- `git push -u origin`

### Branch Deletion
- `git branch -D` (force delete branch)
- `git push --delete origin <branch>`
- `git push origin --delete <branch>`
- `git push origin :<branch>`

### History Rewriting
- `git rebase -i` (interactive rebase)
- `git reset --hard HEAD~<n>`
- `git reset --hard <commit>`
- `git commit --amend` (when pushing to shared branches)
- `git filter-branch`
- `git reflog expire`

### Destructive Operations
- `git clean -fd` (force delete untracked files)
- `git checkout -- .` (discard all changes)
- `git reset --hard` (discard all changes)
- `git rm -rf`

### Configuration Changes
- `git config --global`
- `git config user.name`
- `git config user.email`
- `git remote set-url`
- `git remote add`
- `git remote remove`

### Stash Operations (Potentially Destructive)
- `git stash drop`
- `git stash clear`

## Allowed Git Commands

AI agents MAY safely execute these read-only and non-destructive commands:

### Information Gathering
- `git status`
- `git log`
- `git show`
- `git diff`
- `git branch`
- `git branch -a`
- `git remote -v`
- `git ls-files`

### Safe Local Operations
- `git add <files>`
- `git commit -m "<message>"` (local commits only)
- `git checkout <branch>` (existing branches)
- `git checkout -b <branch>` (new local branches)
- `git stash`
- `git stash list`
- `git stash show`
- `git stash pop` (with caution)

### Safe Pulls
- `git pull` (only if working directory is clean)
- `git fetch`

## Implementation Notes

- These rules should be enforced at the tool/command execution level
- AI agents should verify commands against this list before execution
- When in doubt, prefer read-only operations over potentially destructive ones
- Always confirm with users before executing any git command that modifies repository state
- Local commits are generally safe, but pushing should always require explicit user approval

## Violations

If an AI agent attempts to execute a prohibited command, it should:
1. Refuse to execute the command
2. Explain why the command is dangerous
3. Suggest safer alternatives when appropriate
4. Ask the user to execute the command manually if truly needed

## Last Updated

Created: 2025-08-14