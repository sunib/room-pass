#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
# This suite owns its cluster. It never reads the user's kubeconfig or reuses a
# running cluster: even policy-removal controls stay confined to this fixture.
cluster=dex-network-e2e
clusters=$(k3d cluster list -o json)
if jq -e 'any(.[]; .name == "dex-network-e2e")' <<< "$clusters" >/dev/null; then
  echo "Cluster $cluster already exists; remove it explicitly before running this suite." >&2
  exit 1
fi
mkdir -p .local
cleanup() {
  k3d cluster delete "$cluster"
  rm -f .local/network-kubeconfig
}
trap cleanup EXIT
k3d cluster create "$cluster" --image rancher/k3s:v1.31.5-k3s1 --servers 1 --agents 1 --wait \
  --kubeconfig-update-default=false --kubeconfig-switch-context=false \
  --k3s-arg '--disable=traefik@server:0'
k3d kubeconfig get "$cluster" > .local/network-kubeconfig
chmod 600 .local/network-kubeconfig
if [ -f /.dockerenv ]; then
  docker network connect "k3d-$cluster" "$(hostname)"
  trap 'docker network disconnect "k3d-$cluster" "$(hostname)" || true; cleanup' EXIT
  server_ip=$(docker inspect "k3d-$cluster-server-0" --format '{{(index .NetworkSettings.Networks "k3d-dex-network-e2e").IPAddress}}')
  sed -i "s|server: .*|server: https://$server_ip:6443|" .local/network-kubeconfig
fi
# The fixture's Dex keeps its state in Kubernetes and does not create its CRDs.
# Poll for Established rather than `kubectl wait`, which fails at once on a CRD
# the apiserver has not yet written a status for (see test/e2e/up.sh).
kubectl --kubeconfig .local/network-kubeconfig apply -k deploy/dex-crds
for crd in $(kubectl kustomize deploy/dex-crds | sed -n 's/^  name: \(.*\.dex\.coreos\.com\)$/\1/p'); do
  for attempt in $(seq 1 60); do
    status=$(kubectl --kubeconfig .local/network-kubeconfig get crd "$crd" -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>/dev/null || true)
    [ "$status" = True ] && break
    [ "$attempt" -lt 60 ] || { echo "ERROR: CRD $crd was not Established within 120s." >&2; exit 1; }
    sleep 2
  done
done
ROOM_PASS_NETWORK_E2E=1 go test -v -count=1 -timeout=12m ./test/network
