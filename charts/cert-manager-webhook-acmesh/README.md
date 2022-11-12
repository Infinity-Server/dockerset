### cert-manager-webhook-acmesh

> A cert-manager webhook built on top of acme.sh

> Status: WIP

### Usage

> Install

```bash
# add infinity-server repo
helm repo add infinity-server https://infinity-server.github.io/dockerset
# update repo
helm repo update
# install chart
helm upgrade --install \
  cert-manager-webhook-acmesh infinity-server/cert-manager-webhook-acmesh \
  --namespace cert-manager \
  --create-namespace \
  -f ./cert-manager-webhook-acmesh.values.yml
```

```yaml
# cert-manager-webhook-acmesh.values.yml, use dnspod(dns_dp) for example
clusterIssuer:
  name: acmesh-dnspod
  staging: false
  enabled: true
  ttl: 600
  dnsapi: "dns_dp" # same as acme.sh --dns parameter
  env: # same as acme.sh dnsapi environments
    - "DP_Id=123456"
    - "DP_Key=ajsdhflasjhdflahsd"
```

> Usage

```yaml
---
# secret for receive certificate contents
apiVersion: v1
kind: Secret
metadata:
  name: tls-certs
  namespace: cert-manager
type: kubernetes.io/tls
data:
  tls.crt: ""
  tls.key: ""

# a certificate resource
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: dnspod-demo-certificate
  namespace: cert-manager
spec:
  secretName: tls-certs
  issuerRef:
    kind: ClusterIssuer
    name: acmesh-dnspod
  dnsNames:
    - "test.dosk.host"
    - "*.test.dosk.host"
```
