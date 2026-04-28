# Repository Management (autonomous agents)

This document explains how Argo CD `Repository` secrets are synchronized between the principal (control plane) and agents (workload clusters).

## Overview

In Argo CD Agent, Git repository credentials are stored as Kubernetes Secrets with the label `argocd.argoproj.io/secret-type: repository`. Repository management varies by agent mode:

- **Managed agents**: Repositories are created on the control plane and distributed to agents that need them.
- **Autonomous agents**: Repositories are created and managed locally on the workload cluster. The agents will not sync them back to the control plane.

### Autonomous Agent Mode

In autonomous mode, repositories are created and managed **locally on the workload cluster**. Repository credentials remain completely isolated to each agent cluster with no synchronization to the principal.

### Creating Repositories for Autonomous Agents

Repository secrets are created directly in the argocd installation namespace on the autonomous agent cluster. These repositories are immediately available to local Argo CD Applications and do not require project scoping for basic functionality.

### Local Repository Management

Autonomous agents handle repository secrets entirely within their local cluster:

1. **Local Creation**: Repository secrets are created directly on the agent cluster
2. **Immediate Availability**: Repositories are immediately usable by local Argo CD Applications
3. **No Distribution**: Repositories remain isolated to the specific agent cluster
4. **Independent Management**: Each agent manages its own set of repository credentials

### Example: Creating a Repository on an Autonomous Agent

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: frontend-repo
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
type: Opaque
stringData:
  type: git
  url: https://github.com/myorg/frontend-app.git
  username: deploy-user
  password: ghp_xyz789token
  # Note: project field optional for autonomous agents
  # Only needed if associating with local AppProjects
```

### Local Project Association

While not required for basic functionality, repositories on autonomous agents can still be associated with local AppProjects:

```yaml
stringData:
  # ... other repository fields
  project: local-frontend-project  # References local AppProject
```

### Repository Lifecycle in Autonomous Mode

- **Creation**: Create repository secrets directly on the agent cluster
- **Updates**: Modify repository secrets on the agent cluster; changes take effect immediately
- **Deletion**: Delete repository secrets on the agent cluster; Applications using the repository will lose access
- **Isolation**: Repository changes on one autonomous agent do not affect other agents or the principal

### Security Considerations for Autonomous Agents

Since repository credentials remain local to each agent cluster:

1. **Credential Isolation**: Each agent can use different credentials for the same repository
2. **Independent Rotation**: Repository credentials can be rotated independently on each agent
3. **Local RBAC**: Repository access is controlled entirely by local Kubernetes RBAC
4. **No Central Visibility**: Principal cluster has no visibility into autonomous agent repository configurations

!!! note "Repository Independence"
    Repository credentials on autonomous agents are completely independent. The same repository URL can use different credentials on different agent clusters.

For more information about Argo CD Agent configuration and other features, see:

- [Agent Configuration Reference](../../configuration/reference/agent.md)
- [Principal Configuration Reference](../../configuration/reference/principal.md)
- [Application synchronization (managed agents)](../managed-agent/applications-managed-mode.md)
- [Application synchronization (autonomous agents)](applications-autonomous-mode.md)
- [AppProjects (managed agents)](../managed-agent/appprojects-managed-mode.md)
- [AppProjects (autonomous agents)](appprojects-autonomous-mode.md)
