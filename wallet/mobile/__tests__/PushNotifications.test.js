import PushNotification from 'react-native-push-notification';
import {
  initNotifications,
  sendLocalNotification,
  disableNotifications,
  checkPermissions,
  requestPermissions,
  createDefaultChannel,
} from '../src/services/PushNotifications';

jest.mock('react-native-push-notification', () => {
  const createChannel = jest.fn((_, cb) => cb(true));
  const requestPermissions = jest.fn(() =>
    Promise.resolve({alert: true, authorizationStatus: 2}),
  );
  const localNotification = jest.fn();
  const cancelAllLocalNotifications = jest.fn();
  const removeAllDeliveredNotifications = jest.fn();
  const checkPermissions = jest.fn(cb => cb({alert: true, sound: true}));
  const abandonPermissions = jest.fn(() => Promise.resolve());
  return {
    createChannel,
    requestPermissions,
    localNotification,
    cancelAllLocalNotifications,
    removeAllDeliveredNotifications,
    checkPermissions,
    abandonPermissions,
  };
});

describe('PushNotifications service', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('initializes by creating a channel and requesting permissions', async () => {
    const result = await initNotifications();
    expect(PushNotification.createChannel).toHaveBeenCalledTimes(1);
    expect(PushNotification.requestPermissions).toHaveBeenCalledTimes(1);
    expect(result).toBe(true);
  });

  it('creates default channel explicitly', async () => {
    await createDefaultChannel();
    expect(PushNotification.createChannel).toHaveBeenCalledWith(
      expect.objectContaining({
        channelId: 'paw-default',
        channelName: 'PAW Notifications',
      }),
      expect.any(Function),
    );
  });

  it('sends local notification after ensuring channel', async () => {
    await sendLocalNotification({
      title: 'Test',
      message: 'Body',
      data: {txHash: 'abc'},
    });
    expect(PushNotification.localNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        channelId: 'paw-default',
        title: 'Test',
        message: 'Body',
        userInfo: {txHash: 'abc'},
      }),
    );
  });

  it('disables notifications by abandoning permissions and clearing queues', async () => {
    await disableNotifications();
    expect(PushNotification.cancelAllLocalNotifications).toHaveBeenCalled();
    expect(
      PushNotification.removeAllDeliveredNotifications,
    ).toHaveBeenCalled();
  });

  it('checks permissions and resolves to true when any notification type is enabled', async () => {
    const result = await checkPermissions();
    expect(PushNotification.checkPermissions).toHaveBeenCalled();
    expect(result).toBe(true);
  });

  it('interprets requestPermissions response correctly', async () => {
    const allowed = await requestPermissions();
    expect(allowed).toBe(true);
  });
});
