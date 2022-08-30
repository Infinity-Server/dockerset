### frpc-ingress

> frp client crd for kubernetes

### Usage

- Server: setup frps on any server in anyway as your wish, eg:

```ini
[common]
bind_port = 7000
token = 12345678
```

- Client(Kubernetes)

> Install

```shell
helm repo add infinity-server https://infinity-server.github.io/dockerset
helm repo update
helm upgrade --install frpc-ingress infinity-server/frpc-ingress 
```

> Deploy


```yaml
# common config
---
apiVersion: crds.dosk.host/v1alpha1
kind: FRPCIngress
metadata:
  name: common
spec:
  kind: Config
  config:
    - server_port = 7000        # frps server port
    - server_addr = 1.1.1.1     # frps server address
    - token = 12345678          # some other configs you need
```

```yaml
# rule example, a service is required
---
apiVersion: v1
kind: Service
metadata:
  name: demo-service
  labels:
    app: demo-service
spec:
  ports:
    - port: 53
      name: dns
      protocol: UDP
    - port: 8000
      name: http
      protocol: TCP
  selector:
    app: demo-service

---
apiVersion: crds.dosk.host/v1alpha1
kind: FRPCIngress
metadata:
  name: demo-service-dns-ingress
spec:
  kind: Rule
  service:
    name: demo-service
    port: 53
    protocol: UDP
    remotePort: 53

---
apiVersion: crds.dosk.host/v1alpha1
kind: FRPCIngress
metadata:
  name: demo-service-http-ingress
spec:
  kind: Rule
  service:
    name: demo-service
    port: 8000
    protocol: TCP
    remotePort: 8000
    # extraConfig:
    #   - foo = baz
```
