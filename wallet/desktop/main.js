const { app, BrowserWindow, ipcMain, dialog, Menu, shell } = require('electron');
const path = require('path');
const { autoUpdater } = require('electron-updater');
const Store = require('electron-store');

// Initialize secure storage
const store = new Store({
  name: 'paw-wallet-config',
  encryptionKey: 'paw-blockchain-secure-key-2025'
});

let mainWindow;
let isDev = process.env.NODE_ENV === 'development';

// Single instance lock
const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
  app.quit();
} else {
  app.on('second-instance', () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
  });
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    title: 'PAW Wallet',
    icon: path.join(__dirname, 'build', 'icon.png'),
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      enableRemoteModule: false,
      preload: path.join(__dirname, 'preload.js'),
      sandbox: true
    },
    backgroundColor: '#1a1b26',
    show: false
  });

  // Load app
  if (isDev) {
    mainWindow.loadURL('http://localhost:3000');
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, 'dist', 'index.html'));
  }

  // Show window when ready
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  // Create menu
  createMenu();

  // Handle window close
  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Security: Prevent navigation
  mainWindow.webContents.on('will-navigate', (event, url) => {
    if (!url.startsWith('http://localhost:3000') && !url.startsWith('file://')) {
      event.preventDefault();
    }
  });

  // Security: Prevent new windows
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });
}

function createMenu() {
  const template = [
    {
      label: 'File',
      submenu: [
        {
          label: 'New Wallet',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow.webContents.send('menu-action', 'new-wallet');
          }
        },
        {
          label: 'Import Wallet',
          accelerator: 'CmdOrCtrl+I',
          click: () => {
            mainWindow.webContents.send('menu-action', 'import-wallet');
          }
        },
        {
          label: 'Backup Wallet',
          accelerator: 'CmdOrCtrl+B',
          click: () => {
            mainWindow.webContents.send('menu-action', 'backup-wallet');
          }
        },
        { type: 'separator' },
        {
          label: 'Settings',
          accelerator: 'CmdOrCtrl+,',
          click: () => {
            mainWindow.webContents.send('menu-action', 'settings');
          }
        },
        { type: 'separator' },
        { role: 'quit' }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' },
        { role: 'selectAll' }
      ]
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'forceReload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' }
      ]
    },
    {
      label: 'Wallet',
      submenu: [
        {
          label: 'Send',
          accelerator: 'CmdOrCtrl+S',
          click: () => {
            mainWindow.webContents.send('menu-action', 'send');
          }
        },
        {
          label: 'Receive',
          accelerator: 'CmdOrCtrl+R',
          click: () => {
            mainWindow.webContents.send('menu-action', 'receive');
          }
        },
        {
          label: 'Transaction History',
          accelerator: 'CmdOrCtrl+H',
          click: () => {
            mainWindow.webContents.send('menu-action', 'history');
          }
        },
        { type: 'separator' },
        {
          label: 'Address Book',
          accelerator: 'CmdOrCtrl+A',
          click: () => {
            mainWindow.webContents.send('menu-action', 'address-book');
          }
        }
      ]
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'Documentation',
          click: async () => {
            await shell.openExternal('about:blank');
          }
        },
        {
          label: 'Report Issue',
          click: async () => {
            await shell.openExternal('about:blank');
          }
        },
        { type: 'separator' },
        {
          label: 'Check for Updates',
          click: () => {
            checkForUpdates(true);
          }
        },
        { type: 'separator' },
        {
          label: 'About PAW Wallet',
          click: () => {
            dialog.showMessageBox(mainWindow, {
              type: 'info',
              title: 'About PAW Wallet',
              message: 'PAW Desktop Wallet',
              detail: `Version: ${app.getVersion()}\nElectron: ${process.versions.electron}\nChrome: ${process.versions.chrome}\nNode: ${process.versions.node}\n\nCopyright © 2025 PAW Blockchain`
            });
          }
        }
      ]
    }
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// IPC Handlers
ipcMain.handle('store:get', async (event, key) => {
  return store.get(key);
});

ipcMain.handle('store:set', async (event, key, value) => {
  store.set(key, value);
  return true;
});

ipcMain.handle('store:delete', async (event, key) => {
  store.delete(key);
  return true;
});

ipcMain.handle('store:clear', async () => {
  store.clear();
  return true;
});

ipcMain.handle('dialog:showOpenDialog', async (event, options) => {
  return dialog.showOpenDialog(mainWindow, options);
});

ipcMain.handle('dialog:showSaveDialog', async (event, options) => {
  return dialog.showSaveDialog(mainWindow, options);
});

ipcMain.handle('dialog:showMessageBox', async (event, options) => {
  return dialog.showMessageBox(mainWindow, options);
});

ipcMain.handle('app:getVersion', async () => {
  return app.getVersion();
});

ipcMain.handle('app:getPath', async (event, name) => {
  return app.getPath(name);
});

// Auto-updater
function checkForUpdates(showNoUpdateDialog = false) {
  if (isDev) {
    if (showNoUpdateDialog) {
      dialog.showMessageBox(mainWindow, {
        type: 'info',
        title: 'Updates',
        message: 'Auto-updates are disabled in development mode'
      });
    }
    return;
  }

  autoUpdater.checkForUpdatesAndNotify();

  autoUpdater.on('update-available', () => {
    dialog.showMessageBox(mainWindow, {
      type: 'info',
      title: 'Update Available',
      message: 'A new version is available. It will be downloaded in the background.',
      buttons: ['OK']
    });
  });

  autoUpdater.on('update-downloaded', () => {
    dialog.showMessageBox(mainWindow, {
      type: 'info',
      title: 'Update Ready',
      message: 'A new version has been downloaded. Restart the application to apply the updates.',
      buttons: ['Restart', 'Later']
    }).then((result) => {
      if (result.response === 0) {
        autoUpdater.quitAndInstall();
      }
    });
  });

  if (showNoUpdateDialog) {
    autoUpdater.on('update-not-available', () => {
      dialog.showMessageBox(mainWindow, {
        type: 'info',
        title: 'No Updates',
        message: 'You are running the latest version.',
        buttons: ['OK']
      });
    });
  }
}

// App lifecycle
app.whenReady().then(() => {
  createWindow();

  // Check for updates on startup (after 3 seconds)
  setTimeout(() => {
    checkForUpdates(false);
  }, 3000);

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// Security: Disable hardware acceleration on some systems
if (process.platform === 'linux') {
  app.disableHardwareAcceleration();
}

// Handle uncaught exceptions
process.on('uncaughtException', (error) => {
  console.error('Uncaught Exception:', error);
  dialog.showErrorBox('Application Error', error.message);
});

process.on('unhandledRejection', (error) => {
  console.error('Unhandled Rejection:', error);
});
