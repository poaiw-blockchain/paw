import React, {useEffect, useState} from 'react';
import {View, Text, FlatList, StyleSheet} from 'react-native';
import WalletService from '../services/WalletService';

const TransactionsScreen = () => {
  const [transactions, setTransactions] = useState([]);

  useEffect(() => {
    const load = async () => {
      const txs = await WalletService.getTransactions();
      setTransactions(txs);
    };

    load();
  }, []);

  return (
    <View style={styles.container}>
      {transactions.length === 0 ? (
        <Text style={styles.empty}>No transactions found</Text>
      ) : (
        <FlatList
          data={transactions}
          keyExtractor={item => item.txhash || item.hash || String(item.height)}
          renderItem={({item}) => (
            <View style={styles.txRow}>
              <Text style={styles.txHash}>{item.txhash || 'Tx'}</Text>
              <Text style={styles.txInfo}>
                {item.height ? `Height ${item.height}` : 'Pending'}
              </Text>
            </View>
          )}
        />
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0b0b0f',
    padding: 16,
  },
  empty: {
    color: '#888',
    textAlign: 'center',
    marginTop: 32,
  },
  txRow: {
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#222',
  },
  txHash: {
    color: '#fff',
    fontFamily: 'Courier',
  },
  txInfo: {
    color: '#777',
    marginTop: 4,
  },
});

export default TransactionsScreen;
