import React, {useState} from 'react';
import {
  View,
  Text,
  TextInput,
  StyleSheet,
  TouchableOpacity,
  Alert,
} from 'react-native';
import WalletService from '../services/WalletService';

const ImportWalletScreen = ({navigation}) => {
  const [mnemonic, setMnemonic] = useState('');
  const [walletName, setWalletName] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleImport = async () => {
    try {
      setIsSubmitting(true);
      await WalletService.importWallet({
        mnemonic,
        walletName: walletName.trim() || 'Imported Wallet',
        password,
      });
      Alert.alert('Wallet imported', 'Wallet imported successfully', [
        {text: 'OK', onPress: () => navigation.replace('Home')},
      ]);
    } catch (error) {
      Alert.alert('Error', error.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.label}>Mnemonic Phrase</Text>
      <TextInput
        style={[styles.input, styles.textArea]}
        placeholder="Enter 12 or 24 word mnemonic"
        placeholderTextColor="#666"
        multiline
        value={mnemonic}
        onChangeText={setMnemonic}
      />

      <Text style={styles.label}>Wallet Name</Text>
      <TextInput
        style={styles.input}
        placeholder="Imported Wallet"
        placeholderTextColor="#666"
        value={walletName}
        onChangeText={setWalletName}
      />

      <Text style={styles.label}>Encryption Password</Text>
      <TextInput
        style={styles.input}
        placeholder="Enter a strong password"
        placeholderTextColor="#666"
        secureTextEntry
        value={password}
        onChangeText={setPassword}
      />

      <TouchableOpacity
        style={[styles.button, isSubmitting && styles.buttonDisabled]}
        onPress={handleImport}
        disabled={isSubmitting}>
        <Text style={styles.buttonText}>
          {isSubmitting ? 'Importing...' : 'Import Wallet'}
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
  textArea: {
    height: 120,
    textAlignVertical: 'top',
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

export default ImportWalletScreen;
