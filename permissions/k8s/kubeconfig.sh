
SERVER="$(
  kubectl config view --raw --flatten --output \
    jsonpath='{.clusters[0].cluster.server}'
)"

AUTHORITY="$(
  kubectl config view --raw --flatten --output \
    jsonpath='{.clusters[0].cluster.certificate-authority-data}'
)"

SECRET="$(
  kubectl get serviceaccount task1 --output \
    jsonpath="{.secrets[0].name}"
)"

TOKEN="$(
  kubectl get secret "$SECRET" --output \
    jsonpath="{.data.token}" | base64 --decode
)"

(
  export KUBECONFIG="$(mktemp)"
  {
    kubectl config set-cluster cluster --server="https://$SERVER"
    kubectl config set clusters.cluster.certificate-authority-data "$AUTHORITY"
    kubectl config set-credentials task1 --token="$TOKEN"
    kubectl config set-context cluster --cluster=cluster --user=task1
    kubectl config use-context cluster
  } > /dev/null
  cat "$KUBECONFIG" && rm "$_"
)