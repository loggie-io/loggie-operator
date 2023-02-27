# Loggie Operator

> English | [中文](./README_cn.md)

Loggie operator is an optional supplementary and extensible component of Loggie. The current main functions are:

- Automatically inject loggie sidecar and collect Pod container logs

## Deploy and run

### Local

1. Deploy or run a Kubernetes cluster

2. Generate certificate and MutatingWebhookConfiguration configuration

   The certificate is used by the webhook server that is automatically injected into the sidecar, please execute:
    ```
    LOCAL_IP=${ip}
    ./cert/generate_cert_local.sh --hostname ${LOCAL_IP}
    ```
   The LOCAL_IP should be set to the local IP address, and the network connectivity from the Kubernetes APIServer to the IP needs to be ensured.

   This step generates:

    - Certificates and keys, etc., are placed in the /tmp/cert directory by default
    - MutatingWebhookConfiguration configuration: you can use `kubectl get mutatingwebhookconfiguration` to view

3. Run: `make run`


## Usage

### Automatically inject loggie sidecar

**Pre-check:**

- Please ensure that in the config.yml configuration of the operator, `sidecar.enabled` is set to true, and the injected loggie image and system configuration meet the requirements. (deployed using helm chart, the configuration is in the values.yml file)
- If the loggie DaemonSet is deployed in the Kubernetes cluster at the same time, in order to ensure that the LogConfig/clusterLogConfig created for injection into the sidecar does not affect the collection of the DaemonSet, please upgrade the Loggie DaemonSet version to v1.5+. This version of Loggie will ignore the `sidecar.loggie .io/inject: "true"` annotation for LogConfig/ClusterLogConfig.


**Create Configuration:**

1. Create a LogConfig/ClusterLogConfig with `sidecar.loggie.io/inject: "true"` annotation.

    - The log collection configuration created here can be understood as the log collection configuration injected into the Pod Loggie sidecar.
    - This configuration is the same as DaemonSet, the only difference is that the annotation must be present, otherwise it will not be recognized by the Loggie Operator.

2. Add `sidecar.loggie.io/inject: "true"` annotation to the pod template of the DaemonSet/StatefulSet used for collection.

   Please note:
    - The annotation needs to be added to the pod template of DaemonSet/StatefulSet, not the annotation of DaemonSet/StatefulSet itself. Only in this way, the created Pod will have the annotation.
    - After the DaemonSet/StatefulSet is newly created or the Pod is rebuilt, the Loggie Sidecar will be automatically injected

**Example:**


1. Create a LogConfig with `sidecar.loggie.io/inject: "true"` annotations

eg:
```
cat << EOF | kubectl apply -f -
apiVersion: loggie.io/v1beta1
kind: LogConfig
metadata:
   annotations:
     sidecar.loggie.io/inject: "true"
   name: tomcat
   namespace: default
spec:
   pipeline:
     sink: |
       type: dev
       printEvents: true
       codec:
         pretty: true
     sources: |
       - type: file
         name: common
         paths:
           - /usr/local/tomcat/logs/**
   selector:
     labelSelector:
       app: tomcat
     type: pod
EOF
```

2. Create a tomcat deployment with `sidecar.loggie.io/inject: "true"` annotations:

eg:
```
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
    name: tomcat
    namespace: default
spec:
    replicas: 1
    selector:
       matchLabels:
          app: tomcat
    template:
       metadata:
          annotations:
             sidecar.loggie.io/inject: "true"
          labels:
             app: tomcat
       spec:
          containers:
             - image: tomcat
               name: tomcat
EOF
```

3. Verify:

    - Is Sidecar injection normal: `kubectl get pod`, tomcat contains two containers
    - Check whether there is a collection log, you can check it through `kubectl logs -f ${pod-name} loggie`

**Other**

Please note:

- The Loggie Sidecar automatic injection form cannot collect the stdout log of the business container. If you want to collect the stdout log of the business container, you need the business container to transfer the stdout log to a log file for Loggie to collect.