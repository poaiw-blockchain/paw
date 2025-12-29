import React, {useState, useEffect} from 'react';
import {View, Text, TextInput, StyleSheet, TouchableOpacity, Alert, Modal} from 'react-native';
import {Buffer} from 'buffer';
import { signLedgerSend } from '../services/hardware/flows';
import KeyStore from '../services/KeyStore';
import PawAPI from '../services/PawAPI';
import BiometricAuth from '../services/BiometricAuth';
import {buildAndSignSendTx} from '../utils/transaction';

const SendScreen = () => {
  const [address, setAddress] = useState('');
  const [amount, setAmount] = useState('');
  const [fromAddress, setFromAddress] = useState('');
  const [useHardware, setUseHardware] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [biometricEnabled, setBiometricEnabled] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState(false);
  const [password, setPassword] = useState('');
  const [pendingSignDoc, setPendingSignDoc] = useState(null);

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
        // Software signing - authenticate and sign with stored private key
        const txParams = {
          fromAddress,
          toAddress: address,
          amount: (Number(amount) * 1_000_000).toString(),
          chainId,
          accountNumber,
          sequence,
        };

        if (biometricEnabled) {
          // Use biometric authentication
          try {
            await BiometricAuth.authenticate('Authenticate to sign transaction');
            // After biometric auth, we still need the password to decrypt the wallet
            // Store params and show password modal
            setPendingSignDoc(txParams);
            setShowPasswordModal(true);
            return; // handlePasswordSubmit will complete the transaction
          } catch (bioError) {
            // Fall back to password
            setPendingSignDoc(txParams);
            setShowPasswordModal(true);
            return;
          }
        } else {
          // Direct password entry
          setPendingSignDoc(txParams);
          setShowPasswordModal(true);
          return; // handlePasswordSubmit will complete the transaction
        }
      }
    } catch (err) {
      Alert.alert('Send failed', err.message || 'Unable to sign transaction');
    } finally {
      setIsSending(false);
    }
  };

  const handlePasswordSubmit = async () => {
    if (!password || !pendingSignDoc) {
      Alert.alert('Error', 'Please enter your password');
      return;
    }

    setShowPasswordModal(false);
    setIsSending(true);

    try {
      // Retrieve wallet using password
      const wallet = await KeyStore.retrieveWallet(password);
      if (!wallet || !wallet.privateKey) {
        throw new Error('Unable to decrypt wallet. Check your password.');
      }

      // Build and sign the transaction
      const signedTx = buildAndSignSendTx({
        ...pendingSignDoc,
        privateKey: wallet.privateKey,
      });

      // Broadcast the transaction
      const result = await PawAPI.broadcastTransaction(
        Buffer.from(JSON.stringify(signedTx.tx)).toString('base64'),
        'BROADCAST_MODE_SYNC',
      );

      if (result && result.txhash) {
        Alert.alert(
          'Transaction Sent',
          `Transaction hash:\n${result.txhash.substring(0, 20)}...`,
          [{text: 'OK', onPress: () => {
            setAddress('');
            setAmount('');
          }}],
        );
      } else {
        Alert.alert('Transaction Submitted', 'Transaction was submitted to the network');
      }
    } catch (err) {
      Alert.alert('Transaction Failed', err.message || 'Unable to sign or broadcast transaction');
    } finally {
      setIsSending(false);
      setPassword('');
      setPendingSignDoc(null);
    }
  };

  const handlePasswordCancel = () => {
    setShowPasswordModal(false);
    setPassword('');
    setPendingSignDoc(null);
    setIsSending(false);
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

      {/* Password Modal for Software Signing */}
      <Modal
        visible={showPasswordModal}
        transparent={true}
        animationType="fade"
        onRequestClose={handlePasswordCancel}>
        <View style={styles.modalOverlay}>
          <View style={styles.modalContent}>
            <Text style={styles.modalTitle}>Enter Password</Text>
            <Text style={styles.modalSubtitle}>
              Enter your wallet password to sign this transaction
            </Text>
            <TextInput
              style={styles.modalInput}
              placeholder="Password"
              placeholderTextColor="#666"
              value={password}
              onChangeText={setPassword}
              secureTextEntry={true}
              autoFocus={true}
            />
            <View style={styles.modalButtons}>
              <TouchableOpacity
                style={styles.modalButtonCancel}
                onPress={handlePasswordCancel}>
                <Text style={styles.modalButtonCancelText}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={styles.modalButtonConfirm}
                onPress={handlePasswordSubmit}>
                <Text style={styles.modalButtonConfirmText}>Sign</Text>
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>
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
  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.7)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  modalContent: {
    backgroundColor: '#1a1a24',
    borderRadius: 12,
    padding: 24,
    width: '85%',
    maxWidth: 320,
  },
  modalTitle: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 8,
    textAlign: 'center',
  },
  modalSubtitle: {
    color: '#888',
    fontSize: 14,
    marginBottom: 20,
    textAlign: 'center',
  },
  modalInput: {
    backgroundColor: '#15151d',
    borderRadius: 8,
    padding: 12,
    color: '#fff',
    marginBottom: 20,
    borderWidth: 1,
    borderColor: '#333',
  },
  modalButtons: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  modalButtonCancel: {
    flex: 1,
    paddingVertical: 12,
    marginRight: 8,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#555',
    alignItems: 'center',
  },
  modalButtonCancelText: {
    color: '#888',
    fontWeight: '600',
  },
  modalButtonConfirm: {
    flex: 1,
    paddingVertical: 12,
    marginLeft: 8,
    borderRadius: 8,
    backgroundColor: '#4A90E2',
    alignItems: 'center',
  },
  modalButtonConfirmText: {
    color: '#fff',
    fontWeight: '600',
  },
});

export default SendScreen;
