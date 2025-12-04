import React from 'react';
import {createStackNavigator} from '@react-navigation/stack';
import WelcomeScreen from '../screens/WelcomeScreen';
import HomeScreen from '../screens/HomeScreen';
import CreateWalletScreen from '../screens/CreateWalletScreen';
import ImportWalletScreen from '../screens/ImportWalletScreen';
import SendScreen from '../screens/SendScreen';
import ReceiveScreen from '../screens/ReceiveScreen';
import TransactionsScreen from '../screens/TransactionsScreen';

const AuthStack = createStackNavigator();
const MainStack = createStackNavigator();

const screenOptions = {
  headerStyle: {backgroundColor: '#121212'},
  headerTintColor: '#fff',
  headerTitleStyle: {fontWeight: '600'},
};

const AuthStackScreen = () => (
  <AuthStack.Navigator screenOptions={screenOptions}>
    <AuthStack.Screen
      name="Welcome"
      component={WelcomeScreen}
      options={{headerShown: false}}
    />
    <AuthStack.Screen
      name="CreateWallet"
      component={CreateWalletScreen}
      options={{title: 'Create Wallet'}}
    />
    <AuthStack.Screen
      name="ImportWallet"
      component={ImportWalletScreen}
      options={{title: 'Import Wallet'}}
    />
  </AuthStack.Navigator>
);

const MainStackScreen = () => (
  <MainStack.Navigator screenOptions={screenOptions}>
    <MainStack.Screen
      name="Home"
      component={HomeScreen}
      options={{title: 'PAW Wallet'}}
    />
    <MainStack.Screen
      name="Send"
      component={SendScreen}
      options={{title: 'Send Tokens'}}
    />
    <MainStack.Screen
      name="Receive"
      component={ReceiveScreen}
      options={{title: 'Receive Tokens'}}
    />
    <MainStack.Screen
      name="Transactions"
      component={TransactionsScreen}
      options={{title: 'Transactions'}}
    />
  </MainStack.Navigator>
);

const AppNavigator = ({isAuthenticated}) => {
  return isAuthenticated ? <MainStackScreen /> : <AuthStackScreen />;
};

export default AppNavigator;
