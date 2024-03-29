const fs = require('fs');
const k8s = require('./k8s');

const kubeConfig = new k8s.KubeConfig();
kubeConfig.loadFromDefault();
const watch = new k8s.Watch(kubeConfig);
const allConfigs = new Map();

const makeSection = (frpcConfig, key) => {
  let content = '';
  content += `[${key}]\n`;
  content += frpcConfig.get(key).join('\n');
  content += '\n\n';
  return content;
};

const extractConfigName = (name) => {
  return name || 'default';
};

watch.getOnce(
    '/apis/crds.dosk.host/v1alpha1/frpc-ingresses', {},
    (result) => {
      let items = [];
      try {
        items = JSON.parse(result).items;
      } catch (e) {
        console.error('[ERRO] get items failed ...');
        process.exit();
      }
      for (const item of items) {
        if (item.spec.kind === 'Config') {
          allConfigs.set(extractConfigName(item.metadata.name), new Map());
        }
      }
      for (const item of items) {
        const name = item.metadata.name || 'undefined';
        const namespace = item.metadata.namespace || 'default';
        const frpcSection =
            (item.spec.kind === 'Config') ? 'common' : `${name}@${namespace}`;
        const frpcConfigName = (item.spec.kind === 'Config') ?
            item.metadata.name :
            (item.spec.service && item.spec.service.targetConfig || '');
        const frpcConfig = allConfigs.get(extractConfigName(frpcConfigName));
        if (frpcConfig.has(frpcSection)) {
          console.error(`[ERRO] duplicated section ${frpcSection}, remain the new one ...`);
        }
        switch (item.spec.kind) {
          case 'Config': {
            if (!item.spec.config || item.spec.config.length === 0) {
              console.error('[ERRO] empty config detected, ignore it ...');
            } else {
              frpcConfig.set(frpcSection, item.spec.config);
            }
            break;
          }
          case 'Rule': {
            if (!item.spec.service) {
              console.error('[ERRO] empty rule, ignore it ...');
              break;
            }
            const type = item.spec.service.protocol ?
                item.spec.service.protocol.toLowerCase() :
                'tcp';
            const extraConfig = item.spec.service.extraConfig || [];
            const svcName = item.spec.service.name || 'default';
            const svcNamespace = item.spec.service.namespace || 'default';
            const configSection = [
              `type = ${type}`,
              `local_ip = ${svcName}.${svcNamespace}.svc.cluster.local`,
              `local_port = ${item.spec.service.port}`,
            ];
            if (item.spec.service.remotePort) {
              configSection.push(
                  `remote_port = ${item.spec.service.remotePort}`)
            }
            if (item.spec.service.customDomains) {
              configSection.push(
                  `custom_domains = ${item.spec.service.customDomains}`)
            }
            if (item.spec.service.subdomain) {
              configSection.push(`subdomain = ${item.spec.service.subdomain}`)
            }
            frpcConfig.set(frpcSection, [...configSection, ...extraConfig]);
            break;
          }
        }
        if (!frpcConfig.has('common')) {
          fs.writeFileSync(
              `/frp/client/${extractConfigName(frpcConfigName)}.ini`, '',
              {encoding: 'utf-8'});
          continue;
        }
        let configContent = makeSection(frpcConfig, 'common');
        for (const key of frpcConfig.keys()) {
          if (key !== 'common') {
            configContent += makeSection(frpcConfig, key);
          }
        }
        fs.writeFileSync(
            `/frp/client/${extractConfigName(frpcConfigName)}.ini`,
            configContent, {encoding: 'utf-8'});
      }
    },
    (err) => {
      console.error(`[ERRO] exit with err: ${err}`);
      process.exit();
    });
