# Docker Image Set

> 只是一些列镜像，一般跑在我的 Raspberry Pi 4B 和我的 NAS 服务器上

## Usage

> 推荐配置，只写必要参数和推荐参数

### caddy

> 开启了命名规则嵌套这个特性

```bash
docker run \
  --network host \
  --volume /etc/caddy/Caddyfile:/etc/caddy/Caddyfile \
  sprnghack/caddy run -config /etc/caddy/Caddyfile
```

### cloudflare_ddns

> 为了通过域名找到机器，只获取本地 IP

```bash
docker run \
  --env API_TOKEN=1234 \
  --env INTERFACE=eth0 \
  --env TIMER_INTERVAL=300
  --env API_ROOT_DOMAIN=dosk.host \
  --env API_TARGET_DOMAIN_4=target.dosk.host \
  springhack/cloudflare_ddns
```

### cockpit_machines

> 最初是为了在树莓派跑虚拟机

```bash
docker run \
  --publish 9090:9090 \
  springhack/cockpit_machines
```

### firefox

> 方便不在家访问内网

```bash
docker run \
  --env PASSWORD=sksks \
  --env GEOMETRY=1770x920 \
  --publish 5070:5070 \
  springhack/firefox
```

### frp

> 紧急情况内网穿透用，frpc/frps 可选

```bash
docker run \
  --volume=/frp/frpc.ini:/frp/frpc.ini \
  pringhack/frp frpc -c /frp/frpc.ini
docker run \
  --volume=/frp/frps.ini:/frp/frps.ini \
  pringhack/frp frps -c /frp/frps.ini
```

### jellyfin

> 魔改自官方 jellyfin/jellyfin 镜像所以原版的参数也可用，增加了 PUID/PGID 支持和 Intel QVS 支持

```bash
docker run \
  --privileged \
  --env PGID=100 \
  --env PUID=1001 \
  --publish 8096:8096 \
  --volume /cache:/cache \
  --volume /media:/media \
  --volume /config:/config \
  --device /dev/dri:/dev/dri \
  springhack/jellyfin
```

### linux_dash

> 一开始为了做监控，后来觉的难用就不用了

```bash
docker run \
  --publish 80:80 \
  springhack/linux_dash
```

### octoprint

> 家里的 3D 打印机用，增加了一个 frps 服务为了让内网的 3D 打印机连上来，内置网络连接插件

```bash
docker run \
  --publish 80:80 \
  --publish 7000:7000 \ # frps 的端口
  --volume /octoprint:/octoprint \
  springhack/octoprint
```

### owapp

> 没啥好说的，玩物下载，取自 Synology DSM7 版本

```bash
docker run \
  --publish 8283:8283 \
  --volume /volume1/OWDownload:/volume1/OWDownload \
  springhack/owapp
```

### pi_dashboard

> 在用的简单监控面板，够用

```bash
docker run \
  --network host \
  --env LISTEN=0.0.0.0:12345 \
  springhack/pi_dashboard
```

### pi_openwrt

> 树莓派跑的 Raspberry Pi OS，OpenWrt 跑在 Docker 里接管网络，详情可以看我的博客

```bash
docker run \
  --privileged \
  --network host \
  --restart always \
  springhack/openwrt
```

### qbittorrent

> 跑 PT，也在树莓派上，增加了一套移动端的主题并自动识别更换，其余参数和官方一致

```bash
docker run \
  --env WEB_PORT=18080 # 新端口，自动切主题并且不限制同源等
  --env WEBUI_PORT=8080 # 老端口，可以不开
  --publish 6881:6881 \
  --publish 6881:6881/udp \
  --publish 8080:8080 \
  --publish 18080:18080 \
  --volume /config:/config \
  --volume /downloads:/downloads \
  springhack/qbittorrent
```

### shell

> 方便在 QTS 里操作 Shell 甚至 Docker

```bash
docker run \
  --pid host \
  --privileged \
  --network host \
  springhack/shell
```

### ssserver

> 懂得都懂，Pyhton 版本

```bash
docker run \
  --env TIMEOUT=300 \
  --env PASSWORD=sksks \
  --env FASTOPEN=false \
  --env SERVER_PORT=7999 \
  --env METHOD=aes-256-cfb \
  --env SERVER_ADDR=0.0.0.0 \
  --publish 7999:7999 \
  springhack/ssserver
```

### xunlei

> 迅雷的 UOS 版本，Xpra 作为 WebUI

```bash
docker run \
  --env PGID=100 \
  --env PUID=1001 \
  --env PRA_PASSWORD=sksk \
  --publish 7010:7010 \
  --volume /home/dosk/downloads:/home/dosk/downloads \
  springhack/xunlei
```

### xunlei_cgi

> 迅雷 Synology DSM7 版本，需要 SYS_PTRACE 权限，因为用到了 proot

```bash
docker run \
  --publish 5050:5050 \
  --cap-add SYS_PTRACE \
  --volume /downloads:/downloads \
  springhack/xunlei_cgi
```

### zerotier

> 异地组网，深得我心

```bash
docker run \
  --network host \
  --cap-add NET_ADMIN \
  --device /dev/net/tun:/dev/net/tun \
  --volume /var/lib/zerotier-one:/var/lib/zerotier-one \
```

