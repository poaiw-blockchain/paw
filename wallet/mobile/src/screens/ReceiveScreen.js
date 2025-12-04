import React, {useEffect, useState} from 'react';
import {View, Text, StyleSheet} from 'react-native';
import WalletService from '../services/WalletService';

const ReceiveScreen = () => {
  const [address, setAddress] = useState('');

  useEffect(() => {
    const load = async () => {
      const info = await WalletService.getWalletInfo();
      setAddress(info.address || '');
    };
    load();
  }, []);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Receive PAW</Text>
      <Text style={styles.subtitle}>Your wallet address</Text>
      <Text style={styles.address}>{address || 'No wallet configured'}</Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#0b0b0f',
    padding: 16,
  },
  title: {
    color: '#fff',
    fontSize: 22,
    fontWeight: '600',
    marginBottom: 8,
  },
  subtitle: {
    color: '#aaa',
    marginBottom: 16,
  },
  address: {
    color: '#4A90E2',
    fontSize: 16,
    textAlign: 'center',
    fontFamily: 'Courier',
  },
});

export default ReceiveScreen;
