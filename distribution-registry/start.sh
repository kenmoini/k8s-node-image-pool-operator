#!/bin/sh

# The ConfigMap mounts all the configuration files into /cacheConsumers/mirror-config
# We then need to get the NodeName and copy to the /etc/containers/registries.conf.d/99-node-image-pool.conf
# so that Podman/Buildah/CRI-O can use it when pulling images

cat /cacheConsumers/mirror-config/$MY_NODE_NAME > /etc/containers/registries.conf.d/99-node-image-pool.conf

echo "Starting registry with custom entrypoint on node $MY_NODE_NAME ..."

/entrypoint.sh /etc/distribution/config.yml