# AI Agent Rules and Instructions

This directory contains safety rules and guidelines for working with AI agents in this repository.

## For Engineers

When working with AI agents (Claude Code, GitHub Copilot, etc.) on this repository, please follow these guidelines:

### 1. Safety Rules Compliance

**ALWAYS** instruct AI agents to read and follow the safety rules before performing any operations:

- Reference the `ai-agent-rules/git-safety-rules.md` file in your prompts
- Ensure agents understand prohibited git commands
- Verify agents will ask for permission before executing potentially dangerous operations

### 2. Best Practices

#### Initial Setup
- Start conversations by directing agents to review safety rules
- Provide context about the repository structure and conventions
- Set clear boundaries on what operations are allowed

#### During Development
- Monitor agent actions for compliance with safety rules
- Review all git commands before execution
- Never allow agents to push to remote repositories without explicit approval
- Require manual confirmation for any history-rewriting operations

#### Code Changes
- Ensure agents follow existing code conventions
- Review all file modifications before committing
- Test changes thoroughly before merging

### 3. Emergency Procedures

If an AI agent violates safety rules or performs unauthorized operations:

1. **Immediately stop** the agent's execution
2. **Assess damage** - check git status and recent commits
3. **Rollback if necessary** - use `git reset` or `git revert` as appropriate
4. **Report the incident** - document what happened and update safety rules if needed

### 4. Repository-Specific Guidelines

This is the **insights-results-smart-proxy** repository:
- Uses Go programming language
- Has established code conventions and testing procedures
- Requires careful handling of configuration files
- Has CI/CD pipelines that must not be broken

### 5. File Structure

```
ai-agent-rules/
├── README.md                 # This instructions file
├── git-safety-rules.md       # Git command safety rules
└── example-prompts.md         # Example prompts for common tasks
```

## Quick Reference

### Safe Commands for Agents
- `git status`, `git log`, `git diff` (information gathering)
- `git add`, `git commit` (local changes only)
- `git checkout -b` (new branches)
- File read/write operations
- Code analysis and suggestions

### Prohibited Commands for Agents
- `git push` (any variant)
- `git reset --hard`
- `git rebase -i`
- `git branch -D`
- Configuration changes
- Force operations

## Support

If you have questions about AI agent safety or need to update these rules:
- Review existing safety violations or incidents
- Discuss with the team before making changes
- Update documentation when new patterns emerge

---

**Remember**: These safety rules exist to protect the repository and maintain code quality. Following them ensures productive and safe AI-assisted development.