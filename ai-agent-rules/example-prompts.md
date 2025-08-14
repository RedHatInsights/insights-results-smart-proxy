# Example Prompts for AI Agents

This file contains example prompts that engineers can use when working with AI agents on this repository. All prompts include references to safety rules.

## Initial Setup Prompt

Use this prompt at the beginning of any AI agent session:

```
You are working on the insights-results-smart-proxy repository. Before performing any operations:

1. READ and FOLLOW the safety rules in `ai-agent-rules/git-safety-rules.md`
2. You MUST NOT execute any git commands listed as prohibited in those rules
3. Always ask for explicit permission before executing git push, force operations, or any destructive commands
4. Focus on safe, local operations and code analysis

Please confirm you understand these safety requirements before proceeding with any tasks.

The repository is a Go-based service. Examine the existing code structure and follow established conventions.
```

