/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2025-01-09 10:44:37
 *  Filename: www/js/index.js
 *  Description: Created by SpringHack using vim automatically.
 */
const { createRoot } = ReactDOM;
const { useRef, useState, useEffect, createElement } = React;
const { Spinner, Button, Form, Badge, Figure, Modal } = ReactBootstrap;

const app = document.querySelector('#app');
const root = createRoot(app);

const COUNT_PER_PAGE = 100;
const SEARCH_PER_PAGE = 5;

const openEditor = async (id) => {
  if (!id) {
    id = await createDocument();
  }
  window.open(`editor.html?id=${encodeURIComponent(id)}`, id);
};

const removeDocument = async (id, event) => {
  event.stopPropagation();
  const yes = confirm('Delete this document ?');
  if (yes) {
    await deleteDocument(id);
    location.reload();
  }
};

const Nav = () => {
  const visibleRef = useRef(null);
  const [list, setList] = useState([]);
  const [isLoading, setLoading] = useState(false);
  useEffect(() => {
    const observer = new IntersectionObserver((entries) => {
      const [entry] = entries;
      if (entry.isIntersecting) {
        if (isLoading) return;
        setLoading(true);
        setList((oldList) => {
          getList(oldList.length, COUNT_PER_PAGE).then((items) => {
            setList([].concat(oldList, items));
            setLoading(false);
          });
          return oldList;
        });
      }
    }, {
      root: null,
      threshold: 0.5,
      rootMargin: '0px'
    });
    if (visibleRef.current) {
      observer.observe(visibleRef.current);
    }
    return () => {
      if (visibleRef.current) {
        observer.unobserve(visibleRef.current);
      }
    };
  }, [visibleRef]);
  return createElement('div', { className: 'list' }, [
    createElement('div', { className: 'l-group' }, [
      ...list.map((item) => {
        return createElement('div', { className: 'l-group-item', onClick: openEditor.bind(null, item.id) }, [
          createElement('img', { src: 'img/doc.svg' }),
          createElement('div', { className: 'list-title' }, item.name),
          createElement('img', { className: 'list-remove', src: 'img/del.svg', onClick: removeDocument.bind(null, item.id) })
        ])
      })
    ]),
    createElement('div', { className: 'visible-ref', ref: visibleRef }, [
      isLoading
        ? createElement(Spinner, { animation: 'grow' })
        : createElement(Figure.Caption, {}, 'no more')
    ])
  ]);
};

const Search = () => {
  const [keyword, setKeyword] = useState('');
  const [list, setList] = useState([]);
  const [open, setOpen] = useState(false);
  const onChange = (e) => {
    setKeyword(e.currentTarget.value);
  };
  useEffect(() => {
    if (!keyword.trim()) {
      setList([]);
      return;
    }
    searchList(keyword, SEARCH_PER_PAGE).then((items) => {
      setList(items);
    });
  }, [keyword]);
  useEffect(() => {
    if (open) {
      setTimeout(() => {
        document.querySelector('#search-input').focus();
      }, 100);
    }
  }, [open]);
  return createElement('div', { className: 'search-box', onClick: () => setOpen(true) }, [
    createElement('img', { src: 'img/search.svg', className: 'search-title' }),
    createElement(Modal, { show: open, onHide: () => setOpen(false) }, [
      createElement(Modal.Header, {}, [
        createElement(Form.Control, { id: 'search-input', type: 'text', muted: true, onChange })
      ]),
      createElement(Modal.Body, {}, [
        createElement('div', { className: 'search-list l-group' }, [
          ...list.map((item) => {
            return createElement('div', { className: 'l-group-item', onClick: openEditor.bind(null, item.id) }, [
              createElement('img', { src: 'img/doc.svg' }),
              createElement('div', { className: 'list-title' }, item.name),
              createElement('img', { className: 'list-remove', src: 'img/del.svg', onClick: removeDocument.bind(null, item.id) })
            ])
          })
        ])
      ]),
      createElement(Modal.Footer, {}, [
        createElement(Figure.Caption, {}, 'type and search ...')
      ])
    ])
  ]);
};

root.render(
  createElement('div', { className: 'main' }, [
    createElement('div', { className: 'left' }, [
      createElement('div', { className: 'header' }, [
        createElement(Badge, { bg: 'dark', className: 'title' }, 'Excalidraw'),
        createElement(Button, { variant: 'dark', className: 'new', onClick: openEditor.bind(null, null) }, 'NEW')
      ]),
      createElement(Nav, {})
    ]),
    createElement('div', { className: 'right' }, [
      createElement(Search, {})
    ])
  ])
);
