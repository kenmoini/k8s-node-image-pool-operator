# Kubernetes Node Image Pool Operator

## What?

- Targets nodes to act as local caches/mirrors
- Runs a Docker Registry DaemonSet with the host ro mounted /var/lib/containers path
- Targets nodes to use the local caches/mirrors
- Creates a drop-in file in /etc/registries.d/ for the local caches/mirrors by host IP
- Provides node-to-node authentication via ServiceAccounts/OIDC

