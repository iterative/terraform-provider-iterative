#!/bin/bash

SERVER="$(
  kubectl config view --raw --flatten --output \
    jsonpath='{.clusters[0].cluster.server}'
)"

AUTHORITY="$(
  kubectl config view --raw --flatten --output \
    jsonpath='{.clusters[0].cluster.certificate-authority-data}'
)"

SECRET="$(
  kubectl get serviceaccount task --output \
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
    kubectl config set-credentials task --token="$TOKEN"
    kubectl config set-context cluster --cluster=cluster --user=task
    kubectl config use-context cluster
  } >/dev/null
  cat "$KUBECONFIG" && rm "$KUBECONFIG"
)
