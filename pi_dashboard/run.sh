#!/bin/bash

LISTEN=${LISTEN:-80}

# 修改监听端口
echo "PI DASHBOARD LISTENING ON PORT ${LISTEN}"
sed -i -r "s/^(\s+listen.*?)80/\1${LISTEN}/g" /etc/nginx/sites-available/default

# 启动服务
/etc/init.d/php7.4-fpm start
nginx -g "daemon off;"
