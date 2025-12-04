const { contextBridge, ipcRenderer } = require('electron');

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electron', {
  // Store API
  store: {
    get: (key) => ipcRenderer.invoke('store:get', key),
    set: (key, value) => ipcRenderer.invoke('store:set', key, value),
    delete: (key) => ipcRenderer.invoke('store:delete', key),
    clear: () => ipcRenderer.invoke('store:clear')
  },

  // Dialog API
  dialog: {
    showOpenDialog: (options) => ipcRenderer.invoke('dialog:showOpenDialog', options),
    showSaveDialog: (options) => ipcRenderer.invoke('dialog:showSaveDialog', options),
    showMessageBox: (options) => ipcRenderer.invoke('dialog:showMessageBox', options)
  },

  // App API
  app: {
    getVersion: () => ipcRenderer.invoke('app:getVersion'),
    getPath: (name) => ipcRenderer.invoke('app:getPath', name)
  },

  // Menu actions
  onMenuAction: (callback) => {
    ipcRenderer.on('menu-action', (event, action) => callback(action));
  },

  // Remove listener
  removeMenuActionListener: () => {
    ipcRenderer.removeAllListeners('menu-action');
  }
});
