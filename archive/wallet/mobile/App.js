/**
 * PAW Mobile Wallet - Main App Component
 * Handles navigation and app initialization
 */

import React, {useEffect, useState} from 'react';
import {NavigationContainer} from '@react-navigation/native';
import {StatusBar, View, ActivityIndicator, StyleSheet} from 'react-native';
import AppNavigator from './src/navigation/AppNavigator';

// Services
import WalletService from './src/services/WalletService';

const App = () => {
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    initializeApp();
  }, []);

  /**
   * Initialize app and check for existing wallet
   */
  const initializeApp = async () => {
    try {
      // Check if wallet exists
      const hasWallet = await WalletService.hasWallet();
      setIsAuthenticated(hasWallet);
    } catch (error) {
      console.error('Error initializing app:', error);
      setIsAuthenticated(false);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="large" color="#4A90E2" />
      </View>
    );
  }

  return (
    <>
      <StatusBar barStyle="light-content" backgroundColor="#1a1a1a" />
      <NavigationContainer>
        <AppNavigator isAuthenticated={isAuthenticated} />
      </NavigationContainer>
    </>
  );
};

const styles = StyleSheet.create({
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#1a1a1a',
  },
});

export default App;
