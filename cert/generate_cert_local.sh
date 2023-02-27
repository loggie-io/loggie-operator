#!/bin/bash

set -e

CERT_DIR="/tmp/cert"
SERVER_PORT=9443
SECRET=loggie-webhook
NAMESPACE=loggie

usage() {
  cat <<EOF
usage: ${0} [OPTIONS]
The following flags are required.
    --hostname         To deploy in Kubernetes, please use {serviceName}.{namespace}.svc;
                          locally, please use the IP address where the Loggie operator is running locally.

The following flags are optional.
    --namespace        Namespace where webhook service and secret reside. defaults: loggie
    --secret           Secret name for CA certificate and server certificate/key pair. defaults: loggie-webhook
    --cert-dir         The directory where the certificate is stored. defaults: "/tmp/cert"
    --server-port      Server Port. defaults: 9443
EOF
  exit 1
}

while [ $# -gt 0 ]; do
  case ${1} in
      --hostname)
          HOST_NAME="$2"
          shift
          ;;
      --namespace)
          NAMESPACE="$2"
          shift
          ;;
      --secret)
          SECRET="$2"
          shift
          ;;
      --cert-dir)
          CERT_DIR="$2"
          shift
          ;;
      --server-port)
          SERVER_PORT="$2"
          shift
          ;;
      *)
          usage
          ;;
  esac
  shift
done

[ -z "${HOST_NAME}" ] && echo "ERROR: --hostname flag is required" && exit 1


mkdir -p ${CERT_DIR}
cd ${CERT_DIR}
cat > ca-config.json <<EOF
{
  "signing": {
    "default": {
      "expiry": "87600h"
    },
    "profiles": {
      "server": {
        "usages": ["signing", "key encipherment", "server auth", "client auth"],
        "expiry": "87600h"
      }
    }
  }
}
EOF

cat > ca-csr.json <<EOF
{
  "CN": "Kubernetes",
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "C": "China",
      "L": "Hangzhou",
      "O": "Kubernetes",
      "OU": "Kubernetes",
      "ST": "Oregon"
    }
  ]
}
EOF

cfssl gencert -initca ca-csr.json | cfssljson -bare ca

cat > server-csr.json <<EOF
{
  "CN": "admission",
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "C": "China",
      "L": "Hangzhou",
      "O": "Kubernetes",
      "OU": "Kubernetes",
      "ST": "Oregon"
    }
  ]
}
EOF

cfssl gencert \
  -ca=ca.pem \
  -ca-key=ca-key.pem \
  -config=ca-config.json \
  -hostname=${HOST_NAME} \
  -profile=server \
  server-csr.json | cfssljson -bare server

CA_BUNDLE=$(cat ca.pem | base64)

mv ${CERT_DIR}/server-key.pem ${CERT_DIR}/tls.key
mv ${CERT_DIR}/server.pem ${CERT_DIR}/tls.crt

cat > mutating.yaml <<EOF
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: loggie-sidecar-injector-webhook
webhooks:
  - admissionReviewVersions:
      - v1
    clientConfig:
      caBundle: ${CA_BUNDLE}
      url: https://${HOST_NAME}:${SERVER_PORT}/mutate-inject-sidecar
    failurePolicy: Ignore
    matchPolicy: Equivalent
    name: sidecar-injector-webhook.loggie.io
    namespaceSelector: {}
    objectSelector:
      matchExpressions:
      - key: sidecar.loggie.io/inject
        operator: NotIn
        values:
        - "false"

    rules:
      - apiGroups:
          - ""
        apiVersions:
          - v1
        operations:
          - CREATE
          - UPDATE
        resources:
          - pods
        scope: '*'
    sideEffects: None
    timeoutSeconds: 3
EOF

kubectl apply -f ${CERT_DIR}/mutating.yaml