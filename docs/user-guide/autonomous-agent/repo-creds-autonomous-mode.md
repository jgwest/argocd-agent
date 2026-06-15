# Repository Credential Templates (autonomous agents)

This document explains how Argo CD `Repository Credential Templates` (repo-creds) are synchronized between the principal and autonomous agents.

## Overview

Repository credential templates allow you to define credentials once and have them automatically applied to any repository matching a URL pattern. In Argo CD Agent, repo-creds are stored as Kubernetes Secrets with the label `argocd.argoproj.io/secret-type: repo-creds`.

Repo-creds follow the same synchronization model as repository secrets ([managed agents](../managed-agent/repository-managed-mode.md), [autonomous agents](./repository-autonomous-mode.md)):

- **Managed agents**: Repo-creds are created on the control plane and distributed to agents based on AppProject matching.
- **Autonomous agents**: Repo-creds are created and managed locally on the workload cluster.

## Autonomous Agent Mode

In autonomous mode, repositories are created and managed **locally on the workload cluster**. Repository credentials remain completely isolated to each agent cluster with no synchronization to the principal.

### Creating Repositories for Autonomous Agents

Repository secrets are created directly in the argocd installation namespace on the autonomous agent cluster. These repositories are immediately available to local Argo CD Applications and do not require project scoping for basic functionality.

### Example: Creating a Repository on an Autonomous Agent

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-creds
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repo-creds
type: Opaque
stringData:
  type: git
  url: https://github.com/myorg
  username: deploy-user
  password: ghp_xyz789token
  # Note: project field optional for autonomous agents
  # Only needed if associating with local AppProjects
```


## Repo-Creds vs Repository Secrets

| Aspect | Repository Secret | Repo-Creds |
|--------|------------------|------------|
| Label | `secret-type: repository` | `secret-type: repo-creds` |
| Scope | Specific repository URL | URL prefix pattern |
| Use case | Credentials for a single repo | Credentials for all repos under an org or host |

Both types follow identical synchronization and lifecycle behavior.

For more information about Argo CD Agent configuration and other features, see:

- [Agent Configuration Reference](../../configuration/reference/agent.md)
- [Principal Configuration Reference](../../configuration/reference/principal.md)
- [Application synchronization (managed agents)](../managed-agent/applications-managed-mode.md)
- [Application synchronization (autonomous agents)](applications-autonomous-mode.md)
- [AppProjects (managed agents)](../managed-agent/appprojects-managed-mode.md)
- [AppProjects (autonomous agents)](appprojects-autonomous-mode.md)
