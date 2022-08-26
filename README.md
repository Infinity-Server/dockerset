### infinity server helm repo

> [Helm](https://helm.sh) must be installed to use the charts.  Please refer to

> Helm's [documentation](https://helm.sh/docs) to get started.

- Once Helm has been set up correctly, add the repo as follows:

```shell
helm repo add infinity-server https://infinity-server.github.io/dockerset
helm repo update
```

- To install one chart, eg: `frpc-ingress`:

```shell
helm upgrade --install frpc-ingress dosk/frpc-ingress 
```

- To uninstall the chart:

```shell
helm delete frpc-ingress
```
