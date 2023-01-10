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

const extractConfigName = (namespace) => {
  return namespace || 'default';
};

watch.getOnce(
    '/apis/crds.dosk.host/v1alpha1/frpc-ingresses', {},
    (result) => {
      let items = [];
      try {
        items = JSON.parse(result).items;
      } catch (e) {
        console.error('[ERROR] get items failed ...');
        process.exit();
      }
      for (const item of items) {
        if (item.spec.kind === 'Config') {
          allConfigs.set(extractConfigName(item.metadata.namespace), new Map());
        }
      }
      for (const item of items) {
        const frpcConfig = allConfigs.get(extractConfigName(item.metadata.namespace));
        const name = item.metadata.name || 'undefined';
        const namespace = item.metadata.namespace || 'default';
        const frpcSection =
            (item.spec.kind === 'Config') ? 'common' : `${name}@${namespace}`;
        if (frpcConfig.has(frpcSection)) {
          console.error(`[ERROR] duplicated section ${frpcSection} ...`);
          process.exit();
        }
        switch (item.spec.kind) {
          case 'Config': {
            if (!item.spec.config || item.spec.config.length === 0) {
              console.error('[ERROR] empty config ...');
              process.exit();
            }
            frpcConfig.set(frpcSection, item.spec.config);
            break;
          }
          case 'Rule': {
            if (!item.spec.service) {
              console.error('[ERROR] empty rule ...');
              process.exit();
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
              configSection.push(`remote_port = ${item.spec.service.remotePort}`)
            }
            if (item.spec.service.customDomains) {
              configSection.push(`custom_domains = ${item.spec.service.customDomains}`)
            }
            if (item.spec.service.subdomain) {
              configSection.push(`subdomain = ${item.spec.service.subdomain}`)
            }
            frpcConfig.set(frpcSection, [
              ...configSection,
              ...extraConfig
            ]);
            break;
          }
        }
        if (!frpcConfig.has('common')) {
          console.error(`[ERROR] no common config(${extractConfigName(item.metadata.namespace)}) ...`);
          fs.writeFileSync(`/frp/client/${extractConfigName(item.metadata.namespace)}.ini`, '', {encoding: 'utf-8'});
          process.exit();
        }
        let configContent = makeSection(frpcConfig, 'common');
        for (const key of frpcConfig.keys()) {
          if (key !== 'common') {
            configContent += makeSection(frpcConfig, key);
          }
        }
        fs.writeFileSync(`/frp/client/${extractConfigName(item.metadata.namespace)}.ini`, configContent, {encoding: 'utf-8'});
      }
    },
    (err) => {
      console.error(`[ERROR] exit with err: ${err}`);
      process.exit();
    });
