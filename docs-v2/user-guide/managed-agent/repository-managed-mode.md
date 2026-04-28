# Repository Management (managed agents)

This document explains how Argo CD `Repository` secrets are synchronized between the principal (control plane) and agents (workload clusters).

## Overview

In Argo CD Agent, Git repository credentials are stored as Kubernetes Secrets with the label `argocd.argoproj.io/secret-type: repository`. Repository management varies by agent mode:

- **Managed agents**: Repositories are created on the control plane and distributed to agents that need them.
- **Autonomous agents**: Repositories are created and managed locally on the workload cluster. The agents will not sync them back to the control plane.

### Managed Agent Mode

In managed mode, repositories are created on the **control plane** and automatically distributed to workload clusters based on AppProject configuration and agent matching patterns. Only project-scoped repositories are reconciled by the principal.

### Creating Repositories

Repository secrets must be created in the argocd installation namespace on the control plane cluster with proper project association to enable distribution to managed agents.

Requirements for a repository secret to be reconciled by the principal:

- The secret has label `argocd.argoproj.io/secret-type: repository`
- The secret contains a non-empty `project` field in `.data`/`stringData`
- The secret resides in the Argo CD installation namespace (e.g., `argocd`)

### Repository-to-Agent Distribution Logic

The principal distributes a repository secret to a managed agent using a **two-step matching process**:

1. **Project Association**: Repository must include a `project` field referencing an existing AppProject
2. **Pattern Matching**: Agent name must match **both**:
   - At least one destination pattern via `.spec.destinations` (either `name` or a `server` URL that includes `?agentName=<pattern>`; `*` is allowed)  
   - At least one pattern in the AppProject's `.spec.sourceNamespaces` fields

Learn more about AppProject matching logic in the guides for [managed agents](./appprojects-managed-mode.md) and [autonomous agents](../autonomous-agent/appprojects-autonomous-mode.md).

### Example: Repository Distribution Setup

First, create an AppProject that defines which agents should receive repositories:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: backend-services
  namespace: argocd
spec:
  # Agents matching these patterns can source Applications
  sourceNamespaces:
  - "backend-*"
  - "api-*"
  destinations:
  # Applications can deploy to these agent clusters
  - name: "backend-*"
    namespace: "production"
    server: "*"
  - name: "api-*" 
    namespace: "staging"
    server: "*"
  sourceRepos:
  - "https://github.com/myorg/*"
```

Then create a repository secret associated with this project:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backend-repo
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
type: Opaque
stringData:
  type: git
  url: https://github.com/myorg/backend-services.git
  project: backend-services  # Associates with AppProject above
```

This repository will be distributed to agents named:

- `backend-prod` (matches `backend-*` in both sourceNamespaces and destinations)
- `backend-staging` (matches `backend-*` in both sourceNamespaces and destinations)
- `api-production` (matches `api-*` in both sourceNamespaces and destinations)

But **not** to agents like:

- `frontend-prod` (doesn't match any patterns above)
- `ops-eu` (doesn't match any patterns above)

### Agent-Side Repository Processing

When a repository secret is sent to a managed agent, it undergoes processing:

1. **Local Creation**: Repository secret is created in the agent's argocd installation namespace
2. **Metadata Preservation**: Original repository configuration (URL, credentials, etc.) is preserved
3. **Project Context**: Repository remains associated with the original project name
4. **Source UID Tracking**: Agent adds a source UID annotation to track the principal as source
5. **Change Reconciliation**: Manual changes to managed repositories on agents are reverted to match the principal

### Repository Lifecycle in Managed Mode

- **Creation**: Repository secrets created on the principal are automatically sent to matching agents
- **Updates**: Changes to repository secrets on the principal are propagated to all agents that received them
- **Deletion**: Deleting a repository secret on the principal removes it from all agents
- **Project Changes**: Updating AppProject patterns can change which agents receive repositories:
  - New matching agents receive the repository
  - Agents that no longer match will have the repository **removed automatically**

!!! note "Automatic Cleanup"
    When AppProject patterns change and an agent no longer matches, repositories are automatically removed from that agent on the next sync.

!!! note "Pattern Matching"
    Agent matching uses glob patterns and supports both destination `name` and destination `server` (with `?agentName=`). Deny patterns prefixed with `!` are supported.
    Common examples:
    - `*` - matches all agents
    - `prod-*` - matches agents starting with "prod-"
    - `!dev-*` - excludes agents starting with "dev-"

## Troubleshooting

### Repository Not Distributed to Agents (Managed Mode)

If a repository secret created on the principal doesn't appear on managed agents:

#### Check Label, Namespace and Project Association

Ensure the repository:

- Has label `argocd.argoproj.io/secret-type=repository`
- Lives in the Argo CD namespace on the principal (e.g., `argocd`)
- Has a valid, non-empty `project` field:

```bash
# Check if project field exists
kubectl get secret my-repo -n argocd -o jsonpath='{.data.project}' | base64 -d

# If no output, the repository is missing the project field
# Add it using:
kubectl patch secret my-repo -n argocd --type merge -p '{
  "stringData": {
    "project": "my-project-name"
  }
}'
```

#### Verify AppProject Configuration

Check that the AppProject exists and has correct patterns:

```bash
# Check specific AppProject configuration
kubectl get appproject my-project -n argocd -o yaml
```

Verify that:

- The AppProject referenced by the repository exists
- Agent names match patterns in both `.spec.destinations` (either `name` or `server` with `?agentName=`) and `.spec.sourceNamespaces`
- Patterns use correct glob syntax, including any deny patterns (`!pattern`) you intend

#### Check Principal Logs

Look for repository processing messages on the principal:

```bash
kubectl logs -n argocd deployment/argocd-agent-principal
```

#### Verify Agent Connectivity

Ensure agents are connected and processing events:

```bash
kubectl logs -n argocd deployment/argocd-agent
```

#### Stale Repository Data

If agents have outdated repository information:

```bash
# Check source UID annotation on agent
kubectl get secret my-repo -n argocd -o jsonpath='{.metadata.annotations.argocd\.argoproj\.io/source-uid}'

# Restart agent to force resync
kubectl rollout restart -n argocd deployment/argocd-agent
```

### Migration and Recovery

#### Recovering from Principal Loss

If the principal cluster is lost, autonomous agents continue working normally. For managed agents:

1. Repositories remain functional on agents until credentials expire
2. New principal cluster needs repository secrets recreated
3. Agents will reconnect and receive repositories based on AppProject patterns

For more information about Argo CD Agent configuration and other features, see:

- [Agent Configuration Reference](../../configuration/reference/agent.md)
- [Principal Configuration Reference](../../configuration/reference/principal.md)
- [Application synchronization (managed agents)](applications-managed-mode.md)
- [Application synchronization (autonomous agents)](../autonomous-agent/applications-autonomous-mode.md)
- [AppProjects (managed agents)](appprojects-managed-mode.md)
- [AppProjects (autonomous agents)](../autonomous-agent/appprojects-autonomous-mode.md)
