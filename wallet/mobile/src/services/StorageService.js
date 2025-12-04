import AsyncStorage from '@react-native-async-storage/async-storage';

const STORAGE_KEYS = {
  THEME: '@PAW:theme',
  NETWORK: '@PAW:network',
  ADDRESS_BOOK: '@PAW:address_book',
  RECENT_ADDRESSES: '@PAW:recent_addresses',
  PRICE_ALERTS: '@PAW:price_alerts',
};

class StorageService {
  async setItem(key, value) {
    await AsyncStorage.setItem(key, JSON.stringify(value));
    return true;
  }

  async getItem(key, defaultValue = null) {
    const stored = await AsyncStorage.getItem(key);
    if (!stored) {
      return defaultValue;
    }
    try {
      return JSON.parse(stored);
    } catch (error) {
      return defaultValue;
    }
  }

  async removeItem(key) {
    await AsyncStorage.removeItem(key);
    return true;
  }

  async clear() {
    await AsyncStorage.clear();
    return true;
  }

  async setTheme(theme) {
    await this.setItem(STORAGE_KEYS.THEME, theme);
    return true;
  }

  async getTheme() {
    return (await this.getItem(STORAGE_KEYS.THEME, 'dark')) || 'dark';
  }

  async setNetwork(network) {
    await this.setItem(STORAGE_KEYS.NETWORK, network);
    return true;
  }

  async getNetwork() {
    return this.getItem(STORAGE_KEYS.NETWORK, {
      name: 'mainnet',
      rpcUrl: 'http://localhost:1317',
      chainId: 'paw-1',
    });
  }

  async addAddress(entry) {
    const addressBook = (await this.getItem(STORAGE_KEYS.ADDRESS_BOOK, [])) || [];
    const record = {
      ...entry,
      id: entry.id || Date.now().toString(),
      note: entry.note || '',
    };
    const updated = [...addressBook, record];
    await this.setItem(STORAGE_KEYS.ADDRESS_BOOK, updated);
    return record;
  }

  async removeAddress(id) {
    const addressBook = (await this.getItem(STORAGE_KEYS.ADDRESS_BOOK, [])) || [];
    const updated = addressBook.filter(entry => entry.id !== id);
    await this.setItem(STORAGE_KEYS.ADDRESS_BOOK, updated);
    return true;
  }

  async updateAddress(id, updates) {
    const addressBook = (await this.getItem(STORAGE_KEYS.ADDRESS_BOOK, [])) || [];
    const updated = addressBook.map(entry =>
      entry.id === id ? {...entry, ...updates} : entry,
    );
    await this.setItem(STORAGE_KEYS.ADDRESS_BOOK, updated);
    return updated.find(entry => entry.id === id) || null;
  }

  async addRecentAddress(address) {
    const recent =
      (await this.getItem(STORAGE_KEYS.RECENT_ADDRESSES, [])) || [];
    const withoutDup = recent.filter(item => item !== address);
    withoutDup.unshift(address);
    const trimmed = withoutDup.slice(0, 10);
    await this.setItem(STORAGE_KEYS.RECENT_ADDRESSES, trimmed);
    return trimmed;
  }

  async addPriceAlert(alert) {
    const alerts = (await this.getItem(STORAGE_KEYS.PRICE_ALERTS, [])) || [];
    const record = {
      id: alert.id || Date.now().toString(),
      type: alert.type || 'above',
      price: alert.price,
      enabled: true,
    };
    const updated = [...alerts, record];
    await this.setItem(STORAGE_KEYS.PRICE_ALERTS, updated);
    return record;
  }

  async togglePriceAlert(id) {
    const alerts = (await this.getItem(STORAGE_KEYS.PRICE_ALERTS, [])) || [];
    const updated = alerts.map(alert =>
      alert.id === id ? {...alert, enabled: !alert.enabled} : alert,
    );
    await this.setItem(STORAGE_KEYS.PRICE_ALERTS, updated);
    return updated.find(alert => alert.id === id) || null;
  }
}

export default new StorageService();
