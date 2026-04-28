# AppProject Synchronization (autonomous agents)

This guide explains how Argo CD `AppProjects` are synchronized between **autonomous** agents and the principal.

For **managed** agents, see [AppProject synchronization (managed agents)](../managed-agent/appprojects-managed-mode.md). For how modes differ conceptually, see [Agent modes](../../concepts/agent-modes.md).

## Overview

AppProjects in argocd-agent work differently from standard Argo CD deployments. While Applications can be mapped to agents using namespaces, AppProjects require a different synchronization strategy due to their traditional placement in the Argo CD installation namespace.

On autonomous agents, AppProjects are **created on the agent cluster** and **synchronized to the principal**, where they are transformed (including name prefixing) for UI/API visibility and validation.

## Autonomous Agent Mode

### Creating AppProjects

In autonomous mode, AppProjects are created directly on the **agent cluster**. The agent then synchronizes these AppProjects back to the principal.

### Example: Creating an AppProject on an Autonomous Agent

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: my-project
  namespace: argocd
spec:
  destinations:
  - name: "in-cluster"
    namespace: "*"
    server: "https://kubernetes.default.svc"
  # Can also have multiple destinations - all will be transformed
  - name: "another-destination"
    namespace: "specific-ns"
    server: "https://kubernetes.default.svc"
  sourceNamespaces:
  - argocd
  sourceRepos:
  - "*"
```

When this AppProject is created on an autonomous agent named `agent-production`, it will be transformed and appear on the principal as:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: agent-production-my-project  # Prefixed with agent name
  namespace: argocd                  # Placed in Argo CD namespace on principal
spec:
  destinations:
  - name: agent-production           # All destinations point to agent
    namespace: "*"
    server: "*"
  - name: agent-production
    namespace: "specific-ns"         # Original namespace restrictions preserved
    server: "*"
  sourceNamespaces:
  - agent-production                 # Agent's namespace on principal
  sourceRepos:
  - "*"
```

### Principal-Side Transformation

When an AppProject is received from an autonomous agent, the principal applies transformations:

!!! note
    AppProjects from autonomous agents on the control plane are not used for reconciliation (there is no app controller running on the principal for these projects). Instead, they allow the Argo CD API (and UI, CLI) to determine which operations are valid to be performed on Applications that reference these AppProjects.

!!! warning "Important"
    AppProjects that are synced from autonomous agents should not be used by other Applications outside of that agent, as they may change unpredictably when the autonomous agent modifies its local AppProject configuration.

1. **Name Prefixing**: The project name is prefixed with the agent name to avoid conflicts:
```
   Original: my-project
   On Principal: agent-production-my-project
```

2. **Source Namespaces**: Transformed to allow Applications from the agent's namespace on the principal:
```yaml
   sourceNamespaces:
   - agent-production  # The agent's namespace on the principal
```

3. **Destinations**: All destinations are transformed to point at the agent cluster:
```yaml
   destinations:
   - name: agent-production  # The agent name
     server: "*"
     namespace: "*"  # Preserves original namespace restrictions
```

4. **Namespace Mapping**: The project is placed in the Argo CD namespace on the principal (same as where other AppProjects reside)

### Transformation summary (autonomous)

- **Direction**: AppProject flows from agent to principal  
- **Selection**: All AppProjects created on autonomous agents are synchronized
- **Destinations**: All destinations are transformed to point to the agent cluster (name = agent name, server = "*")
- **Source Namespaces**: Replaced with the agent's namespace on the principal
- **Name**: Prefixed with agent name to avoid conflicts

### Lifecycle Management

- **Creation**: Creating an AppProject on the agent automatically creates it on the principal (with prefixed name)
- **Updates**: Changes to AppProjects on the agent are propagated to the principal
- **Deletion**: Deleting an AppProject on the agent removes it from the principal
- **Conflict Resolution**: The prefixed naming prevents conflicts between AppProjects from different autonomous agents

## Best Practices

1. **Use Meaningful Names**: Choose AppProject names that make sense when prefixed with the agent name

2. **Plan for Conflicts**: Remember that the principal will see `{agent-name}-{project-name}`, so plan accordingly

3. **Local Management**: Only create AppProjects that are specific to the autonomous agent's workload

## Troubleshooting

### AppProject Not Appearing on Principal

1. **Check Agent Mode**: Ensure the agent is in autonomous mode
2. **Verify Creation**: Confirm the AppProject was created on the agent cluster
3. **Check Naming**: Look for the prefixed name on the principal (`{agent-name}-{project-name}`)
4. **Review Connectivity**: Ensure the agent can communicate with the principal

### Pattern Matching Issues

1. **Test Patterns**: Use tools like `fnmatch` to test glob patterns
2. **Check Case Sensitivity**: Ensure agent names match the expected case
3. **Verify Wildcards**: Confirm wildcard patterns are correctly specified

## Skip sync label

The skip sync label allows you to prevent specific AppProjects from being synchronized between the principal and agents. This is useful when you want to create AppProjects that should only exist on one side of the synchronization.

### Label Details

- **Label Key**: `argocd-agent.argoproj-labs.io/ignore-sync`
- **Label Value**: `"true"` (must be the exact string "true", case-sensitive)
- **Scope**: Works for both managed and autonomous agent modes

### Preventing Sync to Principal (autonomous mode)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: agent-only-project
  namespace: argocd
  labels:
    argocd-agent.argoproj-labs.io/ignore-sync: "true"  # Skip sync to principal
spec:
  destinations:
  - name: "in-cluster"
    namespace: "*"
    server: "https://kubernetes.default.svc"
  sourceNamespaces:
  - argocd
  sourceRepos:
  - "*"
```

This AppProject will remain only on the autonomous agent cluster and will not be synchronized back to the principal.

### Important Notes

1. **Case Sensitivity**: The label value must be exactly `"true"` (lowercase). Values like `"TRUE"`, `"True"`, `"false"`, or empty strings will **not** trigger the skip sync behavior.

2. **Distribution Override**: In managed mode, the skip sync label overrides the normal distribution logic based on `sourceNamespaces` and `destinations` patterns. Projects with this label will not be sent to any agents.

3. **Label Removal**: If you remove the skip sync label from an existing AppProject, it will begin synchronizing according to the normal rules for your agent mode.

### Use Cases

- **Agent-Only Projects**: Projects specific to local workload cluster requirements
- **Administrative Projects**: Projects used for cluster management that shouldn't be distributed
- **Temporary Isolation**: Preventing sync during maintenance, testing, or gradual rollouts

### Repository Skip Sync Support

The skip sync label also works for **Repository secrets** (Argo CD repository configurations stored as Kubernetes secrets). Repository secrets with the `argocd-agent.argoproj-labs.io/ignore-sync: "true"` label will not be synchronized between the principal and agents.

#### Example: Repository Secret with Skip Sync

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: private-repo
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
    argocd-agent.argoproj-labs.io/ignore-sync: "true"  # Skip sync
data:
  project: <base64-encoded-project-name>
  url: <base64-encoded-repo-url>
  # ... other repository configuration
```

This is useful for repositories that should only be available on specific clusters, such as:

- **Principal-only repositories**: Internal configuration repositories
- **Agent-only repositories**: Cluster-specific private repositories
- **Environment-specific repositories**: Repositories containing sensitive configuration for specific environments

## Security Considerations

- **Autonomous mode**: Agents can create AppProjects locally—enforce RBAC on each workload cluster.

## Monitoring and observability

- **Principal logs**: Observe AppProject events arriving from agents
- **Agent logs**: Local AppProject lifecycle and sync errors
- **Metrics / health checks**: Track upstream sync success
