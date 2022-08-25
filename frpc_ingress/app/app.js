const fs = require('fs');
const k8s = require('./k8s');

const kubeConfig = new k8s.KubeConfig();
kubeConfig.loadFromDefault();
const watch = new k8s.Watch(kubeConfig);
const frpcConfig = new Map();

const makeSection = (key) => {
  let content = '';
  content += `[${key}]\n`;
  content += frpcConfig.get(key).join('\n');
  content += '\n\n';
  return content;
};

watch.getOnce('/apis/dosk.host/v1alpha1/frpc-ingresses', {}, (result) => {
  let items = [];
  try {
    items = JSON.parse(result).items;
  } catch(e) {
    console.error('[ERROR] get items failed ...');
    process.exit();
  }
  for (const item of items) {
    const name = item.metadata.name || '';
    const namespace = item.metadata.namespace || 'default';
    const frpcSection = (item.spec.kind === 'Config')
          ? 'common'
          : `${name}@${namespace}`;
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
        const extraConfig = item.spec.service.extraConfig || [];
        const svcName = item.spec.service.name || 'default';
        const svcNamespace = item.spec.service.namespace || 'default';
        frpcConfig.set(frpcSection, [
          `local_ip = ${svcName}.${svcNamespace}.svc.cluster.local`,
          `local_port = ${item.spec.service.port}`,
          `remote_port = ${item.spec.service.remotePort}`,
          ...extraConfig
        ]);
        break;
      }
    }
  }
  if (!frpcConfig.has('common')) {
    console.error(`[ERROR] no common config ...`);
    fs.writeFileSync('/frp/frpc.ini', '', { encoding: 'utf-8' });
    process.exit();
  }
  let configContent = makeSection('common');
  for (const key of frpcConfig.keys()) {
    if (key !== 'common') {
      configContent += makeSection(key);
    }
  }
  fs.writeFileSync('/frp/frpc.ini', configContent, { encoding: 'utf-8' });
}, (err) => {
  process.exit();
});
