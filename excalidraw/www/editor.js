/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2025-01-08 17:30:57
 *  Filename: editor.js
 *  Description: Created by SpringHack using vim automatically.
 */
const { createRoot } = ReactDOM;
const { useRef, useState, useEffect, createElement } = React;
const { Footer, Sidebar, Excalidraw } = ExcalidrawLib;

const app = document.querySelector('#app');
const root = createRoot(app);
const updateDocumentDebounce = superDebounce(updateDocument, 1000, true);

const url = new URL(location.href);
const id = url.searchParams.get('id') || localStorage.getItem('working_id') || '';
window.name = id;
localStorage.setItem('working_id', id || workingId);
if (!url.searchParams.get('id')) {
  url.searchParams.set('working_id', id);
  location.href = url.toString();
}

const ExcalidrawApp = () => {
  const [name, setName] = useState('');
  const [init, setInit] = useState(null);
  const [excalidrawAPI, setExcalidrawAPI] = useState(null);
  useEffect(() => {
    document.title = name;
  }, [name]);
  useEffect(() => {
    if (!excalidrawAPI) return;
    const libraryUrl = new URLSearchParams(location.hash.slice(1)).get('addLibrary');
    if (!libraryUrl) return;
    const libraryItems = fetch(libraryUrl).then(res => res.blob())
    excalidrawAPI.updateLibrary({
      libraryItems,
      prompt: false,
      merge: true,
      defaultStatus: 'published',
      openLibraryMenu: true
    });
  }, [excalidrawAPI]);
  useEffect(() => {
    getDocument(id).then((doc) => {
      const { name, data: initData } = doc;
      const init = JSON.parse(initData);
      setName(name);
      setInit(init);
    });
  }, []);
  const onChangeName = (_) => {
    const newName = prompt('Change Document Name', name);
    if (!newName) {
      return;
    }
    setName(newName);
    updateDocument(id, newName);
  };
  const onDocumentChange = (elements, appState, files) => {
    const { gridSize, gridStep, gridModeEnabled, viewBackgroundColor } = appState;
    const data = {
      appState: {
        gridSize,
        gridStep,
        gridModeEnabled,
        viewBackgroundColor
      },
      elements,
      files
    };
    updateDocumentDebounce(id, name, JSON.stringify(data));
  };
  return createElement(React.Fragment, {}, [
    createElement('div', { className: 'main', style: { width: '100vw', height: '100vh' } }, [
      init === null
        ? null
        : createElement(Excalidraw, {
          theme: window.matchMedia("(prefers-color-scheme: dark)").matches && ExcalidrawLib.THEME.DARK || ExcalidrawLib.THEME.LIGHT,
          excalidrawAPI: (api) => setExcalidrawAPI(api),
          initialData: init,
          UIOptions: {
            canvasActions: {
              export: false,
              loadScene: false,
              toggleTheme: true
            }
          },
          onChange: onDocumentChange
        }, [
          createElement(Footer, {}, [
            createElement(Sidebar.Trigger, { onToggle: onChangeName, className: 'change-name' }, `· ${name} ·`)
          ])
        ])
    ])
  ]);
};

root.render(createElement(ExcalidrawApp, {}));
