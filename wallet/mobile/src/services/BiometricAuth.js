import ReactNativeBiometrics from 'react-native-biometrics';

class BiometricAuthService {
  constructor() {
    this.rnBiometrics = new ReactNativeBiometrics();
    this.isAvailable = false;
    this.biometryType = null;
  }

  async checkAvailability() {
    try {
      const result = await this.rnBiometrics.isSensorAvailable();
      this.isAvailable = !!result.available;
      this.biometryType = result.biometryType || null;
      return result;
    } catch (error) {
      this.isAvailable = false;
      this.biometryType = null;
      return {available: false, biometryType: null};
    }
  }

  getBiometryTypeName() {
    switch (this.biometryType) {
      case ReactNativeBiometrics.TouchID:
        return 'Touch ID';
      case ReactNativeBiometrics.FaceID:
        return 'Face ID';
      case ReactNativeBiometrics.Biometrics:
        return 'Biometric';
      default:
        return 'Biometric';
    }
  }

  async authenticate(promptMessage = 'Authenticate to continue') {
    const availability = await this.checkAvailability();
    if (!availability.available) {
      throw new Error('Biometrics not available on this device');
    }

    const result = await this.rnBiometrics.simplePrompt({
      promptMessage,
    });

    if (!result.success) {
      throw new Error('Authentication failed');
    }

    return true;
  }

  async createKeys(promptMessage = 'Enable biometric authentication') {
    const availability = await this.checkAvailability();
    if (!availability.available) {
      return {publicKey: null};
    }
    return this.rnBiometrics.createKeys(promptMessage);
  }

  async deleteKeys() {
    const result = await this.rnBiometrics.deleteKeys();
    return result.keysDeleted ?? false;
  }

  async keysExist() {
    const result = await this.rnBiometrics.biometricKeysExist();
    return result.keysExist ?? false;
  }

  async createSignature(promptMessage = 'Sign transaction', payload = '') {
    const availability = await this.checkAvailability();
    if (!availability.available) {
      throw new Error('Biometrics not available on this device');
    }
    const result = await this.rnBiometrics.createSignature({
      promptMessage,
      payload,
    });
    if (!result.success) {
      throw new Error('Signature creation cancelled');
    }
    return result.signature;
  }
}

const BiometricAuth = new BiometricAuthService();
export default BiometricAuth;
