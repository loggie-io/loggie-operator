
# Loggie Operator

Loggie operator为Loggie的一个可选择补充可扩展组件，目前的主要功能有：

- 自动注入loggie sidecar并采集Pod容器日志

## 部署与运行

### 本地

1. 部署或运行Kubernetes集群

2. 生成证书和MutatingWebhookConfiguration配置

   证书供自动注入sidecar的webhook server使用，请执行:
   ```
   LOCAL_IP=${ip}
   ./cert/generate_cert_local.sh --hostname ${LOCAL_IP}
   ```
   其中的LOCAL_IP请设置为本地的IP地址，需要确保从Kubernetes APIServer到该IP的网络联通性。

   该步骤会生成：

   - 证书和密钥等，默认放置在/tmp/cert目录下
   - MutatingWebhookConfiguration配置：可以使用`kubectl get mutatingwebhookconfiguration`查看

3. 运行：`make run`


## 使用

### 自动注入loggie sidecar

**前置检查：**

- 请确保operator的config.yml配置中，sidecar.enabled设置为true，注入的loggie image和系统配置满足需求。（使用helm chart部署，该配置在values.yml文件中）
- 如果Kubernetes集群中同时部署了loggie DaemonSet，为了确保创建用于注入sidecar的LogConfig/clusterLogConfig不影响DaemonSet采集，请升级Loggie DaemonSet版本为v1.5+，该版本的Loggie会忽略带有 `sidecar.loggie.io/inject: "true"` annotation的LogConfig/ClusterLogConfig。


**创建配置：**

1. 创建带有 `sidecar.loggie.io/inject: "true"` annotation的LogConfig/ClusterLogConfig。

   - 这里创建的可以理解为注入Pod Loggie sidecar的日志采集配置。  
   - 该配置和DaemonSet方式一致，唯一区别是必须要有该annotation，否则不会被Loggie Operator识别。

2. 给用于采集的DaemonSet/StatefulSet的pod template上加上 `sidecar.loggie.io/inject: "true"` annotation。

   请注意：
   - 需要将annotation加到DaemonSet/StatefulSet的pod template里，而不是DaemonSet/StatefulSet本身的annotation，只有这样，创建出来的Pod才会带有该annotation。
   - 在DaemonSet/StatefulSet新创建，或者Pod重建后，才会自动注入Loggie Sidecar

**示例：**


1. 创建一个带有`sidecar.loggie.io/inject: "true"` annotations的LogConfig

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

2. 创建一个带有`sidecar.loggie.io/inject: "true"` annotations的tomcat deployment：

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

3. 验证：

   - Sidecar注入是否正常：`kubectl get pod`，tomcat包含有两个container
   - 查看是否有采集日志，可以通过`kubectl logs -f ${pod-name} loggie`查看

**其他**

请注意：

   - Loggie Sidecar自动注入形式，无法采集业务容器的stdout日志，如果要采集业务容器的stdout日志，需要业务容器将stdout日志转输出到一个日志文件里供Loggie采集。