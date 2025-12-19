jest.mock('react-native-ble-plx', () => {
  const writeWithoutResponse = jest.fn().mockResolvedValue(undefined);
  const characteristicsForService = jest.fn().mockResolvedValue([
    { uuid: '0000fef5-0000-1000-8000-00805f9b34fb', serviceUUID: '0000fef4-0000-1000-8000-00805f9b34fb', writeWithoutResponse },
    { uuid: '0000fef6-0000-1000-8000-00805f9b34fb', serviceUUID: '0000fef4-0000-1000-8000-00805f9b34fb' },
  ]);

  const connectedDevice = {
    id: 'mock-ledger',
    discoverAllServicesAndCharacteristics: jest.fn().mockResolvedValue({ characteristicsForService }),
  };

  const device = {
    id: 'mock-ledger',
    connect: jest.fn().mockResolvedValue(connectedDevice),
  };

  const monitorCharacteristicForDevice = jest.fn((_id, _service, _char, cb) => {
    setTimeout(() => cb(null, { value: Buffer.from([0xde, 0xad]).toString('base64') }), 0);
    return { remove: jest.fn() };
  });

  return {
    BleManager: jest.fn().mockImplementation(() => ({
      startDeviceScan: jest.fn((_uuids, _opts, cb) => cb(null, device)),
      stopDeviceScan: jest.fn(),
      monitorCharacteristicForDevice,
      cancelDeviceConnection: jest.fn().mockResolvedValue(undefined),
    })),
  };
});

jest.mock('react-native-biometrics', () => {
  return jest.fn().mockImplementation(() => ({
    simplePrompt: jest.fn().mockResolvedValue({ success: true }),
  }));
});

const TransportBLE = require('../bleTransport.ts').default;

describe('TransportBLE implementation', () => {
  it('connects and exchanges data', async () => {
    const transport = await TransportBLE.create(2000);
    const res = await transport.exchange(Buffer.from([0x00]));
    expect(res.equals(Buffer.from([0xde, 0xad]))).toBe(true);
    await transport.close();
  });
});
