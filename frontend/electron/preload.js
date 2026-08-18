const { contextBridge, ipcRenderer } = require('electron');

// Expose safe, isolated API to the renderer process
contextBridge.exposeInMainWorld('fplAPI', {
  invokeAction: (actionName, payload) => ipcRenderer.invoke('fpl:action', { actionName, payload }),
  getAppStatus: () => ipcRenderer.invoke('app:status'),
  onBackendEvent: (channel, callback) => {
    const validChannels = ['backend:status', 'backend:error'];
    if (validChannels.includes(channel)) {
      ipcRenderer.on(channel, (_event, ...args) => callback(...args));
    }
  }
});
