#!/usr/bin/env bash

if [[ $1 == "--config" ]] ; then
  cat <<EOF
configVersion: v1
onStartup: 0
kubernetes:
  - apiVersion: dosk.host/v1alpha1
    kind: FRPCIngress
    executeHookOnEvent: ["Added","Modified","Deleted"]
EOF
else
  binding=$(jq -r '.[0].binding' ${BINDING_CONTEXT_PATH})
  type=$(jq -r '.[0].type' ${BINDING_CONTEXT_PATH})
  if [[ "$binding" == "onStartup" ]];
  then
    node /app/app.js
    screen -d -m /frp/frpc -c /frp/frpc.ini
  fi
  if [[ "$type" == "Event" ]];
  then
    node /app/app.js
    pkill frpc
    screen -d -m /frp/frpc -c /frp/frpc.ini
  fi
fi
