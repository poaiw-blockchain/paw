import React, {useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  Switch,
  Alert,
} from 'react-native';
import WalletService from '../services/WalletService';

const CreateWalletScreen = ({navigation}) => {
  const [walletName, setWalletName] = useState('');
  const [password, setPassword] = useState('');
  const [useBiometric, setUseBiometric] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleCreate = async () => {
    try {
      setIsSubmitting(true);
      const wallet = await WalletService.createWallet({
        walletName: walletName.trim() || 'My Wallet',
        password,
        useBiometric,
      });
      Alert.alert(
        'Wallet created',
        `Backup this mnemonic:\n${wallet.mnemonic}`,
        [
          {
            text: 'Done',
            onPress: () => navigation.replace('Home'),
          },
        ],
      );
    } catch (error) {
      Alert.alert('Error', error.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.label}>Wallet Name</Text>
      <TextInput
        style={styles.input}
        placeholder="My Wallet"
        placeholderTextColor="#666"
        value={walletName}
        onChangeText={setWalletName}
      />

      <Text style={styles.label}>Password</Text>
      <TextInput
        style={styles.input}
        placeholder="Enter a strong password"
        placeholderTextColor="#666"
        secureTextEntry
        value={password}
        onChangeText={setPassword}
      />

      <View style={styles.toggleRow}>
        <Text style={styles.label}>Enable biometric authentication</Text>
        <Switch
          value={useBiometric}
          onValueChange={setUseBiometric}
          thumbColor={useBiometric ? '#4A90E2' : '#888'}
        />
      </View>

      <TouchableOpacity
        style={[styles.button, isSubmitting && styles.buttonDisabled]}
        disabled={isSubmitting}
        onPress={handleCreate}>
        <Text style={styles.buttonText}>
          {isSubmitting ? 'Creating...' : 'Create Wallet'}
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
    marginBottom: 6,
    marginTop: 12,
  },
  input: {
    backgroundColor: '#15151d',
    borderRadius: 8,
    padding: 12,
    color: '#fff',
  },
  toggleRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 16,
  },
  button: {
    marginTop: 32,
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

export default CreateWalletScreen;
