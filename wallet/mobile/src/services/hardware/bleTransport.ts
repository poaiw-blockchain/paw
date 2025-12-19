import { BleManager, Device } from 'react-native-ble-plx';
import { Platform } from 'react-native';
import ReactNativeBiometrics from 'react-native-biometrics';

const COSMOS_SERVICE_UUID = '0000fef4-0000-1000-8000-00805f9b34fb'; // Ledger Cosmos app service
const LEDGER_WRITE_UUID = '0000fef5-0000-1000-8000-00805f9b34fb';
const LEDGER_NOTIFY_UUID = '0000fef6-0000-1000-8000-00805f9b34fb';
const DEFAULT_TIMEOUT = 60000;

export default class TransportBLE {
  private manager: BleManager;
  private device: Device | null = null;
  private writeCharacteristic: any = null;
  private notifyCharacteristic: any = null;
  private timeout: number;

  constructor(timeout = DEFAULT_TIMEOUT) {
    this.manager = new BleManager();
    this.timeout = timeout;
  }

  static async create(timeout = DEFAULT_TIMEOUT): Promise<TransportBLE> {
    const transport = new TransportBLE(timeout);
    await transport.connect();
    return transport;
  }

  setExchangeTimeout(ms: number) {
    this.timeout = ms;
  }

  async connect(): Promise<void> {
    await this.biometricGate();

    const device = await this.scanAndConnect();
    this.device = device;
    const services = await device.discoverAllServicesAndCharacteristics();
    const chars = await services.characteristicsForService(COSMOS_SERVICE_UUID);
    this.writeCharacteristic = chars.find(c => c.uuid.toLowerCase() === LEDGER_WRITE_UUID);
    this.notifyCharacteristic = chars.find(c => c.uuid.toLowerCase() === LEDGER_NOTIFY_UUID);

    if (!this.writeCharacteristic || !this.notifyCharacteristic) {
      throw new Error('Ledger characteristics not found');
    }
  }

  async exchange(apdu: Buffer): Promise<Buffer> {
    if (!this.writeCharacteristic || !this.notifyCharacteristic) {
      throw new Error('Transport not connected');
    }

    await this.writeCharacteristic.writeWithoutResponse(apdu.toString('base64'));

    const response = await new Promise<Buffer>((resolve, reject) => {
      const sub = this.manager.monitorCharacteristicForDevice(
        this.device!.id,
        this.notifyCharacteristic.serviceUUID,
        this.notifyCharacteristic.uuid,
        (error, characteristic) => {
          if (error) {
            sub.remove();
            reject(error);
            return;
          }
          if (characteristic?.value) {
            sub.remove();
            resolve(Buffer.from(characteristic.value, 'base64'));
          }
        }
      );

      setTimeout(() => {
        sub.remove();
        reject(new Error('BLE exchange timeout'));
      }, this.timeout);
    });

    return response;
  }

  async close(): Promise<void> {
    if (this.device) {
      await this.manager.cancelDeviceConnection(this.device.id).catch(() => {});
    }
    this.device = null;
    this.writeCharacteristic = null;
    this.notifyCharacteristic = null;
  }

  private async scanAndConnect(): Promise<Device> {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.manager.stopDeviceScan();
        reject(new Error('BLE scan timeout'));
      }, this.timeout);

      this.manager.startDeviceScan([COSMOS_SERVICE_UUID], null, async (error, device) => {
        if (error) {
          clearTimeout(timer);
          this.manager.stopDeviceScan();
          reject(error);
          return;
        }

        if (device) {
          clearTimeout(timer);
          this.manager.stopDeviceScan();
          try {
            const connected = await device.connect();
            resolve(connected);
          } catch (err) {
            reject(err);
          }
        }
      });
    });
  }

  private async biometricGate(): Promise<void> {
    try {
      const rnBiometrics = new ReactNativeBiometrics();
      const result = await rnBiometrics.simplePrompt({ promptMessage: 'Authenticate to use Ledger' });
      if (!result.success) {
        throw new Error('Biometric authentication failed');
      }
    } catch (err) {
      throw new Error(`Biometric auth required: ${(err as Error).message}`);
    }
  }

  async requireBiometric(): Promise<void> {
    await this.biometricGate();
  }
}
