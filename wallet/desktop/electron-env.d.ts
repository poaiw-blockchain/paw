interface ElectronStore {
  get: (key: string) => Promise<any>;
  set: (key: string, value: any) => Promise<void>;
  delete: (key: string) => Promise<void>;
  clear: () => Promise<void>;
}

interface ElectronDialog {
  showOpenDialog: (options?: any) => Promise<any>;
  showSaveDialog: (options?: any) => Promise<any>;
  showMessageBox: (options?: any) => Promise<{ response: number }>;
}

interface ElectronApp {
  getVersion: () => Promise<string>;
  getPath: (name: string) => Promise<string>;
}

type MenuActionHandler = (action: string) => void;

interface ElectronAPI {
  store?: ElectronStore;
  dialog?: ElectronDialog;
  app?: ElectronApp;
  onMenuAction?: (handler: MenuActionHandler) => void;
  removeMenuActionListener?: () => void;
}

declare global {
  interface Window {
    electron?: ElectronAPI;
  }
}

export {};
