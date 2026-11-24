'use strict'
// Only loaded into the connection window. Exposes exactly three calls; the password
// is write-only (never returned to any renderer).
const { contextBridge, ipcRenderer } = require('electron')
contextBridge.exposeInMainWorld('helixShell', {
  getConnection: () => ipcRenderer.invoke('helix:get-connection'),
  saveConnection: (v) => ipcRenderer.invoke('helix:save-connection', { serverUrl: v.serverUrl, username: v.username, password: v.password }),
  forgetPassword: () => ipcRenderer.invoke('helix:forget-password'),
})
