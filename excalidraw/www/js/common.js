/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2025-01-10 15:59:46
 *  Filename: www/js/common.js
 *  Description: Created by SpringHack using vim automatically.
 */
const doRequestStorage = async (transaction) => {
  const resp = await fetch('/storage', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      transaction,
      resultFormat: 'map'
    })
  }).then(res => res.json());
  return resp;
};

const createDocument = async () => {
  const id = window.crypto.randomUUID();
  const resp = await doRequestStorage([
    {
      statement: '#CREATE_DOCUMENT',
      values: {
        id,
        name: 'excalidraw-untitled'
      }
    }
  ]);
  return resp?.results?.[0]?.success && id || '';
};

const deleteDocument = async (id) => {
  const resp = await doRequestStorage([
    {
      statement: '#DELETE_DOCUMENT',
      values: {
        id
      }
    }
  ]);
  return resp?.results?.[0]?.success;
};

const getList = async (offset, count) => {
  const resp = await doRequestStorage([
    {
      query: '#GET_LIST',
      values: {
        count,
        offset
      }
    }
  ]);
  return resp?.results?.[0]?.resultSet || [];
};

const searchList = async (keyword, count) => {
  const resp = await doRequestStorage([
    {
      query: '#SEARCH_LIST',
      values: {
        count,
        keyword: `%${keyword}%`
      }
    }
  ]);
  return resp?.results?.[0]?.resultSet || [];
};

const updateDocument = async (id, name, data = null) => {
  const transaction = [
    {
      statement: '#UPDATE_DOCUMENT_NAME',
      values: {
        id,
        name
      }
    }
  ];
  if (data !== null) {
    transaction.pop();
    transaction.push({
      statement: '#UPDATE_DOCUMENT_DATA',
      values: {
        id,
        data
      }
    });
  }
  const resp = await doRequestStorage(transaction);
  return resp?.results?.[0]?.success || false;
};

const getDocument = async (id) => {
  const resp = await doRequestStorage([
    {
      query: '#GET_DOCUMENT',
      values: {
        id
      }
    }
  ]);
  return resp?.results?.[0]?.resultSet?.[0] || null;
};

const funcMap = {
  ignore: false
};

window.addEventListener('beforeunload', () => {
  if (funcMap.ignore) return;
  Object.values(funcMap).forEach(func => func instanceof Function && func.call());
});

document.addEventListener('visibilitychange', () => {
  if (funcMap.ignore) return;
  if (document.hidden) {
    Object.values(funcMap).forEach(func => func instanceof Function && func.call());
  }
});

const superDebounce = (func, delay = 1000, beforeunload = false) => {
  let timer = null;
  const uuid = window.crypto.randomUUID();
  return (...args) => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (beforeunload) {
      funcMap[uuid] = func.bind(null, ...args);
    }
    timer = setTimeout(() => {
      delete funcMap[uuid];
      func.call(null, ...args);
    }, delay);
  };
};
