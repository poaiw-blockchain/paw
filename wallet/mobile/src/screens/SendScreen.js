import React, {useState} from 'react';
import {View, Text, TextInput, StyleSheet, TouchableOpacity, Alert} from 'react-native';
import WalletService from '../services/WalletService';
import PawAPI from '../services/PawAPI';

const SendScreen = () => {
  const [address, setAddress] = useState('');
  const [amount, setAmount] = useState('');
  const [isSending, setIsSending] = useState(false);

  const handleSend = async () => {
    if (!address || !amount) {
      Alert.alert('Missing information', 'Enter recipient and amount');
      return;
    }
    setIsSending(true);
    try {
      // Placeholder implementation for UI consistency
      await new Promise(resolve => setTimeout(resolve, 500));
      Alert.alert('Transaction submitted', 'This is a placeholder implementation');
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
});

export default SendScreen;
