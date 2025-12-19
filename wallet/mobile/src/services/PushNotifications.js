import {Platform} from 'react-native';
import PushNotification from 'react-native-push-notification';

const DEFAULT_CHANNEL_ID = 'paw-default';
const DEFAULT_CHANNEL_NAME = 'PAW Notifications';

let channelReady = false;

export const createDefaultChannel = () =>
  new Promise((resolve, reject) => {
    PushNotification.createChannel(
      {
        channelId: DEFAULT_CHANNEL_ID,
        channelName: DEFAULT_CHANNEL_NAME,
        importance: 4, // high priority
        vibrate: true,
      },
      created => {
        channelReady = true;
        resolve(created);
      },
    );
  });

export const requestPermissions = async () => {
  const perms = await PushNotification.requestPermissions();
  const granted =
    perms === true ||
    perms?.alert === true ||
    perms?.authorizationStatus === 1 ||
    perms?.authorizationStatus === 2;
  return Boolean(granted);
};

export const initNotifications = async () => {
  await createDefaultChannel();
  return requestPermissions();
};

export const sendLocalNotification = async ({
  title,
  message,
  data = {},
} = {}) => {
  if (!channelReady) {
    await createDefaultChannel();
  }
  PushNotification.localNotification({
    channelId: DEFAULT_CHANNEL_ID,
    title,
    message,
    userInfo: data,
    playSound: true,
    soundName: 'default',
    priority: 'high',
  });
};

export const disableNotifications = async () => {
  if (Platform.OS === 'ios') {
    await PushNotification.abandonPermissions();
  }
  PushNotification.cancelAllLocalNotifications();
  PushNotification.removeAllDeliveredNotifications();
};

export const checkPermissions = () =>
  new Promise(resolve => {
    PushNotification.checkPermissions(perms => {
      const enabled =
        perms?.alert === true ||
        perms?.badge === true ||
        perms?.sound === true ||
        perms === true;
      resolve(Boolean(enabled));
    });
  });

export default {
  initNotifications,
  sendLocalNotification,
  disableNotifications,
  checkPermissions,
  createDefaultChannel,
  requestPermissions,
};
