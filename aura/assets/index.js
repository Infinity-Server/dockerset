/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2023-12-29 13:39:45
 *  Filename: assets/index.js
 *  Description: Created by SpringHack using vim automatically.
 */

const data = {
  ...await fetch('/services').then(res => res.json()),
  web: new URL(location.href),
  emoji: {
    start: 0x1f330,
    count: 0x1f353 - 0x1f330 + 1
  }
};

data.items = data.items.map((item) => {
  return {
    ...item,
    order: parseInt(item.metadata.annotations['homelab/order'] || Number.MAX_SAFE_INTEGER)
  };
});

data.items.sort((x, y) => {
  return x.order - y.order;
});

let code = '';
for (const [index, value] of data.items.entries()) {
  if (value.metadata.annotations['homelab/host']) {
    code += `
      <div class=item>
        <a href="${data.web.protocol}//${value.metadata.annotations['homelab/host']}.${data.web.host}">
          <div class=icon>
            ${String.fromCodePoint(index % data.emoji.count + data.emoji.start)}
          </div>
          <div class=name>
            ${value.metadata.annotations['homelab/name'] || ''}
          </div>
        </a>
      </div>
    `;
  }
}

document.getElementById('root').innerHTML = code;
