---
layout: default
---
\[[front page](../overall-architecture.html)\] \[[overall architecture](../overall-architecture.html)\]



# Channel: `rule-content`



## Type

* Directory tree containing YAML files and Markdown documents



## Description

Rule content directory tree contains rules metadata (such us severity or
likelihood) and rules descriptions in structured format (summary, reason etc.).
These information is consumed by Content service for all new rule (content)
release.

## Each rule is stored in its own subtree with the following structure

```
└── {name}
    ├── {error_key}
    │   ├── generic.md
    │   └── metadata.yaml
    ├── more_info.md
    ├── plugin.yaml
    ├── reason.md
    ├── resolution.md
    └── summary.md
```

Please note that `name` and `error_key` are placeholders.

