# Kubernetes Node Image Pool Operator

## What?

- Targets nodes to act as local caches/mirrors
- Runs a Docker Registry DaemonSet with the host ro mounted /var/lib/containers path
- Targets nodes to use the local caches/mirrors
- Creates a drop-in file in /etc/registries.d/ for the local caches/mirrors by host IP
- Provides node-to-node authentication via ServiceAccounts/OIDC

## How it works

- Validates Operand specification
  - Ensures a CachePool and CacheConsumer is defined
  - Ensures there are Nodes that match the CachePool and CacheConsumer selection criteria
- Creates a node-image-pool Namespace
- Creates a ConfigMap in the node-image-pool Namespace for /etc/containers/registries.d/ mirror configuration files with a key per CacheConsumer host as to not include the host itself if it is also a CachePool host.
- Starts a custom Docker Distribution Registry as a DaemonSet on the targeted CachePool hosts
- The entrypoint script copies over the relevant mirror configuration for that host to the hostPath then starts the registry