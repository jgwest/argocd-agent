# AppProject synchronization (managed agents)

This guide explains how Argo CD `AppProjects` are synchronized between the principal (control plane) and **managed** agents.

For **autonomous** agents, see [AppProject synchronization (autonomous agents)](../autonomous-agent/appprojects-autonomous-mode.md). For how modes differ conceptually, see [Agent modes](../../concepts/agent-modes.md).

## Overview

AppProjects in argocd-agent work differently from standard Argo CD deployments. While Applications can be mapped to agents using namespaces, AppProjects require a different synchronization strategy due to their traditional placement in the Argo CD installation namespace.

With managed agents, AppProjects are created on the principal and distributed to agents.

## Managed Agent Mode

### Creating AppProjects

In managed mode, AppProjects must be created on the **principal cluster** (control plane). The principal determines which agents should receive an AppProject by examining two key fields:

1. **`.spec.sourceNamespaces`**: Defines which namespaces can contain Applications using this project
2. **`.spec.destinations`**: Defines which clusters/namespaces Applications can deploy to

### Distribution Logic

The principal distributes an AppProject to a managed agent when **both** conditions are met:

1. The agent name matches one of the patterns in `.spec.destinations[].name`
2. The agent name matches one of the patterns in `.spec.sourceNamespaces`

This uses glob pattern matching, so wildcards like `agent-*` are supported.

### Example: Creating an AppProject for Managed Agents

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: my-project
  namespace: argocd
spec:
  # This project will be distributed to agents matching "agent-*" pattern
  sourceNamespaces:
  - agent-*
  destinations:
  - name: agent-*
    namespace: "guestbook"
    server: "*"
  sourceRepos:
  - "*"
```

When this AppProject is created on the principal, it will be automatically distributed to all connected managed agents whose names match the `agent-*` pattern.

### Agent-Specific Transformation

When an AppProject is sent to an agent, it undergoes transformation to make it agent-specific:

1. **Destinations**: Only destinations matching the agent are kept (using glob pattern matching), and they're transformed to point to the local cluster:
```yaml
   destinations:
   - name: "in-cluster"
     server: "https://kubernetes.default.svc"
     namespace: "guestbook"  # Preserves original namespace restrictions
```

2. **Source Namespaces**: Removed completely since they're only used on the control plane for routing
    - The `sourceNamespaces` field is used on the control plane to determine which agents should receive the AppProject. Once the AppProject arrives at the agent cluster, this field is removed as it's no longer needed.

3. **Roles**: Removed since they're not relevant on the workload cluster

### Lifecycle Management

- **Creation**: When you create an AppProject on the principal, it's automatically distributed to matching agents
- **Updates**: Changes to AppProjects on the principal are propagated to affected agents
- **Deletion**: Deleting an AppProject on the principal removes it from all agents
- **Agent Connection**: When an agent connects, it receives all AppProjects that should be synchronized to it

## Transformation summary (managed)

- **Direction**: AppProject flows from principal to agent
- **Selection**: Uses glob pattern matching on `sourceNamespaces` and `destinations` to determine which agents receive the project
- **Destinations**: Filtered to only include destinations matching the agent, then transformed to `in-cluster`
- **Source Namespaces**: Removed completely since they're only used on the control plane for routing
- **Name**: Remains unchanged


## Best Practices

1. **Use Descriptive Patterns**: Use clear glob patterns in `sourceNamespaces` and `destinations` to target the right agents:
```yaml
   sourceNamespaces:
   - "production-*"
   - "staging-*"
   destinations:
   - name: "production-*"
   - name: "staging-*"
```

2. **Test Connectivity**: Ensure agents are connected before creating AppProjects, or they'll receive them upon next connection

3. **Monitor Distribution**: Check agent logs to verify AppProject distribution is working correctly

## Troubleshooting

### AppProject Not Appearing on Agent

1. **Check Agent Mode**: Ensure the agent is in managed mode
2. **Verify Patterns**: Confirm the agent name matches patterns in `sourceNamespaces` and `destinations`
3. **Check Connectivity**: Verify the agent is connected to the principal
4. **Review Logs**: Check principal and agent logs for synchronization errors

### Pattern Matching Issues

1. **Test Patterns**: Use tools like `fnmatch` to test glob patterns
2. **Check Case Sensitivity**: Ensure agent names match the expected case
3. **Verify Wildcards**: Confirm wildcard patterns are correctly specified

## Skip Sync Label

The skip sync label allows you to prevent specific AppProjects from being synchronized between the principal and agents. This is useful when you want to create AppProjects that should only exist on one side of the synchronization.

### Label Details

- **Label Key**: `argocd-agent.argoproj-labs.io/ignore-sync`
- **Label Value**: `"true"` (must be the exact string "true", case-sensitive)
- **Scope**: Works for both managed and autonomous agent modes

### Preventing AppProject Sync to Agent (Managed Mode)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: principal-only-project
  namespace: argocd
  labels:
    argocd-agent.argoproj-labs.io/ignore-sync: "true"  # Skip sync to agents
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

This AppProject will remain only on the principal cluster and will not be distributed to any agents, regardless of matching patterns in `sourceNamespaces` and `destinations`.

### Important Notes

1. **Case Sensitivity**: The label value must be exactly `"true"` (lowercase). Values like `"TRUE"`, `"True"`, `"false"`, or empty strings will **not** trigger the skip sync behavior.

2. **Distribution Override**: In managed mode, the skip sync label overrides the normal distribution logic based on `sourceNamespaces` and `destinations` patterns. Projects with this label will not be sent to any agents.

3. **Label Removal**: If you remove the skip sync label from an existing AppProject, it will begin synchronizing according to the normal rules for your agent mode.

### Use Cases

- **Principal-Only Projects**: Projects that define permissions for control plane operations
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

- **Managed Mode**: Only the principal can create AppProjects, maintaining central control

## Monitoring and Observability

- **Principal Logs**: Monitor AppProject distribution events
- **Agent Logs**: Watch for AppProject creation/update/deletion events
- **Metrics**: Use available metrics to track AppProject synchronization success/failure rates
- **Health Checks**: Implement monitoring to detect synchronization issues
