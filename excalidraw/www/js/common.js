/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2025-01-09 14:00:45
 *  Filename: js/common.js
 *  Description: Created by SpringHack using vim automatically.
 */
const createDocument = async () => {
  const id = window.crypto.randomUUID();
  const resp = await fetch('/storage', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      resultFormat: 'map',
      transaction: [
        {
          statement: 'INSERT INTO storage (id, name, data) VALUES (:id, :name, :data)',
          values: {
            id,
            name: 'untitled-1',
            data: '{}'
          }
        }
      ]
    })
  }).then(res => res.json());
  if (resp?.results?.[0]?.success) {
    return id;
  }
  return '';
};

const deleteDocument = async (id) => {
  const resp = await fetch('/storage', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      resultFormat: 'map',
      transaction: [
        {
          statement: 'DELETE FROM storage WHERE id = :id',
          values: {
            id
          }
        }
      ]
    })
  }).then(res => res.json());
  return resp?.results?.[0]?.success;
};

const getList = async (offset, count) => {
  const rv = await fetch('/storage', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      resultFormat: 'map',
      transaction: [
        {
          query: 'SELECT id, name from storage ORDER BY time DESC LIMIT :count OFFSET :offset',
          values: {
            count,
            offset
          }
        }
      ]
    })
  }).then(res => res.json());
  return rv?.results?.[0]?.resultSet || [];
};

const searchList = async (keyword, count) => {
  const rv = await fetch('/storage', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      resultFormat: 'map',
      transaction: [
        {
          query: 'SELECT id, name from storage WHERE name like :keyword ORDER BY time DESC LIMIT :count',
          values: {
            count,
            keyword: `%${keyword}%`
          }
        }
      ]
    })
  }).then(res => res.json());
  return rv?.results?.[0]?.resultSet || [];
};

const updateDocument = async (id, name, data = null) => {
  const transaction = [
    {
      statement: `UPDATE storage SET time = CURRENT_TIMESTAMP, name = :name WHERE id = :id`,
      values: {
        id,
        name
      }
    }
  ];
  if (data !== null) {
    transaction.pop();
    transaction.push({
      statement: `UPDATE storage SET time = CURRENT_TIMESTAMP, data = :data WHERE id = :id`,
      values: {
        id,
        data
      }
    });
  }
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
  return resp?.results?.[0]?.success || false;
};

const getDocument = async (id) => {
  const rv = await fetch('/storage', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      resultFormat: 'map',
      transaction: [
        {
          query: 'SELECT * from storage WHERE id = :id',
          values: {
            id
          }
        }
      ]
    })
  }).then(res => res.json());
  return rv?.results?.[0]?.resultSet?.[0] || null;
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
