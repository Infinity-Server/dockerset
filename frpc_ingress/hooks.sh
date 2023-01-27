#!/usr/bin/env bash

hooks::on_config() {
  cat <<EOF
configVersion: v1
onStartup: 0
kubernetes:
  - apiVersion: crds.dosk.host/v1alpha1
    kind: FRPCIngress
    executeHookOnEvent: ["Added","Modified","Deleted"]
EOF
}

hooks::on_startup() {
  rm /frp/client/*
  node /utils/config-generator.js
  for client in $(ls /frp/client);
  do
    screen -d -m /utils/keepalive /frp/frpc -c /frp/client/$client
  done
}

hooks::on_event() {
  pkill -9 screen
  screen -wipe
  rm /frp/client/*
  node /utils/config-generator.js
  for client in $(ls /frp/client);
  do
    screen -d -m /utils/keepalive /frp/frpc -c /frp/client/$client
  done
}

hooks::main() {
  if [[ "$1" == "--config" ]];
  then
    hooks::on_config
  else
    [[ "$(jq -r '.[0].binding' ${BINDING_CONTEXT_PATH})" == "onStartup" ]] \
      && hooks::on_startup || echo ''
    [[ "$(jq -r '.[0].type' ${BINDING_CONTEXT_PATH})" == "Event" ]] \
      && hooks::on_event || echo ''
  fi
}

hooks::main "$@"
