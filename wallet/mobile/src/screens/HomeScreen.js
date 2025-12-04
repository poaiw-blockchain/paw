import React, {useEffect, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  RefreshControl,
} from 'react-native';
import WalletService from '../services/WalletService';

const HomeScreen = () => {
  const [isLoading, setIsLoading] = useState(true);
  const [walletInfo, setWalletInfo] = useState(null);
  const [balance, setBalance] = useState(null);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [info, bal] = await Promise.all([
        WalletService.getWalletInfo().catch(() => null),
        WalletService.getBalance().catch(() => null),
      ]);
      setWalletInfo(info);
      setBalance(bal);
    } catch (error) {
      console.error('Failed to load wallet info', error);
    } finally {
      setIsLoading(false);
      setRefreshing(false);
    }
  };

  const onRefresh = () => {
    setRefreshing(true);
    loadData();
  };

  if (isLoading) {
    return (
      <View style={styles.loadingContainer}>
        <Text style={styles.loadingText}>Loading wallet...</Text>
      </View>
    );
  }

  return (
    <ScrollView
      style={styles.container}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
      }>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Account</Text>
        <Text style={styles.address}>
          {walletInfo?.address || 'No wallet configured'}
        </Text>
        <Text style={styles.walletName}>{walletInfo?.name || 'My Wallet'}</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Balance</Text>
        <Text style={styles.balance}>
          {balance ? `${balance.formatted} ${balance.denom}` : '0.000000 PAW'}
        </Text>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0b0b0f',
    padding: 16,
  },
  loadingContainer: {
    flex: 1,
    backgroundColor: '#0b0b0f',
    alignItems: 'center',
    justifyContent: 'center',
  },
  loadingText: {
    color: '#fff',
    fontSize: 16,
  },
  card: {
    backgroundColor: '#15151d',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  cardTitle: {
    color: '#888',
    textTransform: 'uppercase',
    fontSize: 12,
    letterSpacing: 1.2,
    marginBottom: 8,
  },
  address: {
    color: '#fff',
    fontFamily: 'Courier',
    fontSize: 14,
    marginBottom: 4,
  },
  walletName: {
    color: '#4A90E2',
    fontWeight: '600',
  },
  balance: {
    color: '#fff',
    fontSize: 28,
    fontWeight: '700',
  },
});

export default HomeScreen;
