/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2023-12-29 17:25:24
 *  Filename: assets/k8s.js
 *  Description: Created by SpringHack using vim automatically.
 */
import { getKubernetesInfo } from './k8s-api.js';

const [cpu, pod, mem] = ['cpu', 'pod', 'mem'].map((id) => {
  return document.getElementById(id);
});

const [cpuSpan, podSpan, memSpan] = [cpu, pod, mem].map((elem) => {
  return elem.getElementsByTagName('span')[0];
});

const [cpuFill, podFill, memFill] = [cpu, pod, mem].map((elem) => {
  return elem.querySelector('.progress-fill');
});

const origin = [0, 0];
const target = [100, 100];
const controlA = [0, 70];
const controlB = [100, 30];

const threeBezier = (t, p1, cp1, cp2, p2) => {
  const [x1, y1] = p1;
  const [x2, y2] = p2;
  const [cx1, cy1] = cp1;
  const [cx2, cy2] = cp2;
  const x =
    x1 * (1 - t) * (1 - t) * (1 - t) +
    3 * cx1 * t * (1 - t) * (1 - t) +
    3 * cx2 * t * t * (1 - t) +
    x2 * t * t * t;
  const y =
    y1 * (1 - t) * (1 - t) * (1 - t) +
    3 * cy1 * t * (1 - t) * (1 - t) +
    3 * cy2 * t * t * (1 - t) +
    y2 * t * t * t;
  return [x, y];
};

const refreshK8sInfo = async () => {
  const info = await getKubernetesInfo();
  cpuSpan.textContent = `${info.cpuUsed} / ${info.cpuAvailable} Units`;
  podSpan.textContent = `${info.podsInfo.numReady} / ${info.podsInfo.numItems} Requested`;
  memSpan.textContent = `${info.memoryUsed} / ${info.memoryAvailable} GB`;
  let cpuPercent = info.cpuUsed / info.cpuAvailable;
  let podPercent = info.podsInfo.numReady / info.podsInfo.numItems;
  let memPercent = info.memoryUsed / info.memoryAvailable;
  cpuPercent = threeBezier(cpuPercent, origin, controlA, controlB, target);
  podPercent = threeBezier(podPercent, origin, controlA, controlB, target);
  memPercent = threeBezier(memPercent, origin, controlA, controlB, target);
  cpuFill.style.width = `${100 - cpuPercent[1]}%`;
  podFill.style.width = `${100 - podPercent[1]}%`;
  memFill.style.width = `${100 - memPercent[1]}%`;
}

refreshK8sInfo();
setInterval(refreshK8sInfo, 10 * 1000);
