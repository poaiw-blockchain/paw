import React, {useState, useEffect} from 'react';
import {View, Text, TextInput, StyleSheet, TouchableOpacity, Alert} from 'react-native';
import { signLedgerSend } from '../services/hardware/flows';
import KeyStore from '../services/KeyStore';
import PawAPI from '../services/PawAPI';
import BiometricAuth from '../services/BiometricAuth';

const SendScreen = () => {
  const [address, setAddress] = useState('');
  const [amount, setAmount] = useState('');
  const [fromAddress, setFromAddress] = useState('');
  const [useHardware, setUseHardware] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [biometricEnabled, setBiometricEnabled] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const metadata = await KeyStore.retrieveMetadata();
        if (metadata?.address) {
          setFromAddress(metadata.address);
        }
        if (metadata?.biometricEnabled) {
          setBiometricEnabled(true);
        }
        if (!metadata?.address) {
          const storedAddr = await KeyStore.getAddress();
          if (storedAddr) {
            setFromAddress(storedAddr);
          }
        }
      } catch (err) {
        console.warn('Failed to load wallet metadata', err);
      }
    })();
  }, []);

  const handleSend = async () => {
    if (!fromAddress) {
      Alert.alert('Missing information', 'Set up a wallet before sending');
      return;
    }
    if (!address || !amount) {
      Alert.alert('Missing information', 'Enter recipient and amount');
      return;
    }
    if (amount <= 0) {
      Alert.alert('Invalid amount', 'Amount must be greater than zero');
      return;
    }
    setIsSending(true);
    try {
      const nodeInfo = await PawAPI.getNodeInfo();
      const chainId = nodeInfo?.default_node_info?.network || 'paw-testnet-1';
      const account = await PawAPI.getAccount(fromAddress);
      const baseAccount = account?.base_account || account?.base_vesting_account?.base_account || account;
      const accountNumber = String(baseAccount?.account_number || '');
      const sequence = String(baseAccount?.sequence || '');
      if (!accountNumber || !sequence) {
        throw new Error('Unable to load account metadata (account_number/sequence)');
      }

      const signDoc = {
        chain_id: chainId,
        account_number: accountNumber,
        sequence: sequence,
        fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
        msgs: [
          {
            type: 'cosmos-sdk/MsgSend',
            value: {
              from_address: fromAddress,
              to_address: address,
              amount: [{ denom: 'upaw', amount: (Number(amount) * 1_000_000).toString() }],
            },
          },
        ],
        memo: '',
      };

      if (useHardware) {
        await BiometricAuth.authenticate('Confirm Ledger signing');
        const signed = await signLedgerSend({
          fromAddress,
          toAddress: address,
          amount: (Number(amount) * 1_000_000).toString(),
          accountIndex: 0,
          memo: signDoc.memo,
        });
        Alert.alert('Signed', `Transaction signed with Ledger (BLE) on ${signed.path}`);
      } else {
        // Placeholder software signing; production would derive private key
        Alert.alert('Signed', 'Software signing placeholder');
      }
    } catch (err) {
      Alert.alert('Send failed', err.message || 'Unable to sign transaction');
    } finally {
      setIsSending(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.label}>Recipient</Text>
      <TextInput
        style={styles.input}
        placeholder="paw1..."
        placeholderTextColor="#666"
        value={address}
        onChangeText={setAddress}
      />

      <Text style={styles.label}>Amount</Text>
      <TextInput
        style={styles.input}
        placeholder="0.0"
        placeholderTextColor="#666"
        value={amount}
        onChangeText={setAmount}
        keyboardType="numeric"
      />

      <TouchableOpacity
        style={[styles.button, isSending && styles.buttonDisabled]}
        onPress={handleSend}
        disabled={isSending}>
        <Text style={styles.buttonText}>
          {isSending ? 'Sending...' : 'Send'}
        </Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.secondaryButton, useHardware && styles.secondaryButtonActive]}
        onPress={() => setUseHardware(!useHardware)}
        disabled={isSending}>
        <Text style={styles.secondaryButtonText}>
          {useHardware ? 'Using Ledger (BLE)' : 'Use Ledger (BLE)'}
        </Text>
      </TouchableOpacity>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0b0b0f',
    padding: 16,
  },
  label: {
    color: '#aaa',
    marginTop: 12,
    marginBottom: 6,
  },
  input: {
    backgroundColor: '#15151d',
    borderRadius: 8,
    padding: 12,
    color: '#fff',
  },
  button: {
    marginTop: 24,
    backgroundColor: '#4A90E2',
    paddingVertical: 14,
    borderRadius: 8,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: '#fff',
    fontWeight: '600',
  },
  secondaryButton: {
    marginTop: 12,
    borderColor: '#4A90E2',
    borderWidth: 1,
    paddingVertical: 12,
    borderRadius: 8,
    alignItems: 'center',
  },
  secondaryButtonActive: {
    backgroundColor: '#142846',
  },
  secondaryButtonText: {
    color: '#4A90E2',
    fontWeight: '600',
  },
});

export default SendScreen;
