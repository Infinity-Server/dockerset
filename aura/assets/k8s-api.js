/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2023-12-29 14:39:26
 *  Filename: assets/k8s.js
 *  Description: Created by SpringHack using vim automatically.
 */

const UNITS = ['B', 'K', 'M', 'G', 'T', 'P', 'E'];
const TO_GB = 1024 * 1024 * 1024;
const TO_ONE_CPU = 1000000000;

function parseRam(value) {
  return parseUnitsOfBytes(value);
}

function parseUnitsOfBytes(value) {
  if (!value) return 0;

  const groups = value.match(/(\d+)([BKMGTPEe])?(i)?(\d+)?/) || [];
  const number = parseInt(groups[1], 10);

  // number ex. 1000
  if (groups[2] === undefined) {
    return number;
  }

  // number with exponent ex. 1e3
  if (groups[4] !== undefined) {
    return number * 10 ** parseInt(groups[4], 10);
  }

  const unitIndex = UNITS.indexOf(groups[2]);

  // Unit + i ex. 1Ki
  if (groups[3] !== undefined) {
    return number * 1024 ** unitIndex;
  }

  // Unit ex. 1K
  return number * 1000 ** unitIndex;
}

function parseCpu(value) {
  if (!value) return 0;

  const number = parseInt(value, 10);
  if (value.endsWith('n')) return number;
  if (value.endsWith('u')) return number * 1000;
  if (value.endsWith('m')) return number * 1000 * 1000;
  return number * 1000 * 1000 * 1000;
}

function sumAndFormat(arr) {
  const value = arr.reduce((prev, current) => prev + current, 0);
  return value.toFixed(2);
}

export async function getKubernetesInfo() {
  const [pods, nodes, metrics] = await Promise.all(['pods', 'nodes', 'metrics'].map((endpoint) => {
    return fetch(`/${endpoint}`).then(res => res.json());
  }));

  const cpuUsed = sumAndFormat(metrics.items.map((item) => {
    return parseCpu(item.usage.cpu) / TO_ONE_CPU;
  }));
  const cpuAvailable = sumAndFormat(nodes.items.map((item) => {
    return parseCpu(item.status?.capacity.cpu) / TO_ONE_CPU;
  }));

  const memoryUsed = sumAndFormat(metrics.items.map((item) => {
    return parseRam(item.usage.memory) / TO_GB;
  }));
  const memoryAvailable = sumAndFormat(nodes.items.map((item) => {
    return parseRam(item.status?.capacity.memory) / TO_GB;
  }));

  const podsInfo = {
    numReady: (pods.items || []).filter((pod) => {
      if (pod.status?.phase === 'Succeeded') {
        return true;
      }
      const readyCondition = pod.status?.conditions?.find(condition => condition.type === 'Ready');
      return readyCondition?.status === 'True';
    }).length,
    numItems: (pods.items || []).length
  };

  return {
    cpuUsed, cpuAvailable,
    memoryUsed, memoryAvailable,
    podsInfo
  };
};
