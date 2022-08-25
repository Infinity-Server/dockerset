const fs = require('fs');
const path = require('path');
const https = require('https');
const readline = require('readline');
                                                                         
const K8S_BASEDIR = '/var/run/secrets/kubernetes.io/serviceaccount';
const K8S_TOKENFILE = path.join(K8S_BASEDIR, 'token');
const K8S_CAFILE = path.join(K8S_BASEDIR, 'ca.crt');
                                                                         
const KUBERNETES_SERVICE_HOST = process.env.KUBERNETES_SERVICE_HOST;
const KUBERNETES_SERVICE_PORT = process.env.KUBERNETES_SERVICE_PORT;
                                                                         
class KubeConfig {
  constructor() {
    console.error('New kubetnetes client create');
  }
  loadFromDefault() {
    this._token = fs.readFileSync(K8S_TOKENFILE, { encoding: 'utf-8' });
    this._ca = fs.readFileSync(K8S_CAFILE, { encoding: 'utf-8' });
  }
}
                                                                         
class Watch {
  constructor(kc) {
    this._kc = kc;
    console.error('New kubetnetes watcher create');
  }
  getOnce(path, options = {}, listener, errCallback) {
    https.request({
      path,
      hostname: KUBERNETES_SERVICE_HOST,
      port: KUBERNETES_SERVICE_PORT,
      method: 'GET',
      ca: [ this._kc._ca ],
      headers: {
        'Authorization': `Bearer ${this._kc._token}`,
        'Accept': 'application/json'
      }
    }, (res) => {
      let data = Buffer.from('');
      res.on('data', (chunk) => {
        const newData = Buffer.concat([data, Buffer.from(chunk)]);
        data = newData;
      });
      res.on('end', () => {
        listener(data.toString());
      });
    }).on('error', (err) => errCallback(err)).end();
  }
  watch(path, options = {}, listener, errCallback) {
    https.request({
      hostname: KUBERNETES_SERVICE_HOST,
      port: KUBERNETES_SERVICE_PORT,
      path: `${path}?watch`,
      method: 'GET',
      ca: [ this._kc._ca ],
      headers: {
        'Authorization': `Bearer ${this._kc._token}`,
        'Accept': 'application/json'
      }
    }, (res) => {
      const lines = readline.createInterface({ input: res });
      lines.on('close', () => errCallback('closed'));
      lines.on('line', (line) => {
        try {
          const { type, object } = JSON.parse(line);
          listener(type, object);
        } catch (err) {
          lines.close();
          errCallback(err);
        }
      });
    }).on('error', (err) => errCallback(err)).end();
  }
}
                                                                         
module.exports = {
  Watch,
  KubeConfig
};
