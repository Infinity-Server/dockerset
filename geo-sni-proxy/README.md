# geo-sni-proxy

轻量的 DNS + HTTP/TLS 转发伴侣，支持按 Xray `domains` 语义进行域名匹配并分流。

## 配置项
- `GEO_SNI_PROXY_URI`：上游代理地址，支持 `socks5://`, `socks5h://`, `http://`, `https://`
- `GEO_SNI_PROXY_LOG`：是否开启日志（`true`/`1`/`yes`/`on`/`debug`）
- `GEO_SNI_PROXY_DNS_ADDR`：DNS 监听地址，默认 `127.0.0.1:53`
- `GEO_SNI_PROXY_HTTP_ADDR`：HTTP 监听地址，默认 `127.0.0.1:80`
- `GEO_SNI_PROXY_TLS_ADDR`：TLS 监听地址，默认 `127.0.0.1:443`
- `/metadata/domains.yaml`：域名规则（支持 `ext:file:tag`、`geosite:tag`、`domain:`、`keyword:`、`full:`、`dotless:`、`regexp:`、纯字符串作为 `keyword`）
- `/usr/local/share/xray/geosite.dat`：`geosite` 数据源（如 v2fly/domain-list-community 发布文件）

## 运行
- 本地
  - 准备 `/metadata/domains.yaml` 与 `/usr/local/share/xray/geosite.dat`
  - 设置 `GEO_SNI_PROXY_URI`（例如 `socks5://1.2.3.4:1080`）
  - 可选：配置监听地址与日志开关
  - `go build` 后运行
- Docker
  - `docker build -t geo-sni-proxy .`
  - `docker run -e GEO_SNI_PROXY_URI="socks5://1.2.3.4:1080" -e GEO_SNI_PROXY_LOG=true -p 53:53/udp -p 53:53/tcp -p 80:80 -p 443:443 -v $(pwd)/metadata:/metadata geo-sni-proxy`

## 说明
- 命中规则的 DNS 仅返回 `A=127.0.0.1`（不返回 AAAA），并对 `CNAME/ANY/HTTPS/SVCB` 做安全处理
- HTTP 按 `Host`，TLS 按 `SNI` 进行转发；透明连接通过你配置的上游代理建立
