import React, {useEffect, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  RefreshControl,
  TextInput,
  TouchableOpacity,
  Alert,
} from 'react-native';
import WalletService from '../services/WalletService';
import {
  signLedgerDelegate,
  signLedgerVote,
  signLedgerIbcTransfer,
} from '../services/hardware/flows';

const HomeScreen = () => {
  const [isLoading, setIsLoading] = useState(true);
  const [walletInfo, setWalletInfo] = useState(null);
  const [balance, setBalance] = useState(null);
  const [refreshing, setRefreshing] = useState(false);
  const [validatorAddr, setValidatorAddr] = useState('');
  const [stakeAmount, setStakeAmount] = useState('1');
  const [proposalId, setProposalId] = useState('1');
  const [voteOption, setVoteOption] = useState('VOTE_OPTION_YES');
  const [ibcChannel, setIbcChannel] = useState('channel-0');
  const [ibcReceiver, setIbcReceiver] = useState('');
  const [ibcAmount, setIbcAmount] = useState('1');

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

  const ensureWalletAddress = () => {
    if (!walletInfo?.address) {
      throw new Error('Set up a wallet first');
    }
    return walletInfo.address;
  };

  const handleDelegate = async () => {
    try {
      const delegator = ensureWalletAddress();
      const res = await signLedgerDelegate({
        delegatorAddress: delegator,
        validatorAddress: validatorAddr.trim() || delegator,
        amount: (Number(stakeAmount) * 1_000_000).toString(),
      });
      Alert.alert('Delegation signed', `Ledger path ${res.path || 'm/44/118/0/0/0'} ready to broadcast`);
    } catch (err) {
      Alert.alert('Delegate failed', err.message || 'Unable to sign delegation');
    }
  };

  const handleVote = async () => {
    try {
      const voter = ensureWalletAddress();
      const res = await signLedgerVote({
        voter,
        proposalId: proposalId.trim(),
        option: voteOption || 'VOTE_OPTION_YES',
      });
      Alert.alert('Vote signed', `Ledger path ${res.path || 'm/44/118/0/0/0'} ready to broadcast`);
    } catch (err) {
      Alert.alert('Vote failed', err.message || 'Unable to sign vote');
    }
  };

  const handleIbc = async () => {
    try {
      const sender = ensureWalletAddress();
      if (!ibcReceiver.trim()) {
        throw new Error('Receiver is required');
      }
      const res = await signLedgerIbcTransfer({
        sender,
        receiver: ibcReceiver.trim(),
        sourceChannel: ibcChannel.trim() || 'channel-0',
        amount: (Number(ibcAmount) * 1_000_000).toString(),
      });
      Alert.alert('IBC transfer signed', `Ledger path ${res.path || 'm/44/118/0/0/0'} ready to broadcast`);
    } catch (err) {
      Alert.alert('IBC failed', err.message || 'Unable to sign IBC transfer');
    }
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

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Hardware Flows (Ledger BLE)</Text>
        <Text style={styles.helper}>
          Biometric + hardware signing for common actions. Signs only (no broadcast). Use your validator/proposal/channel details.
        </Text>

        <Text style={styles.sectionLabel}>Delegate</Text>
        <TextInput
          style={styles.input}
          placeholder="Validator address (paw...)"
          placeholderTextColor="#666"
          value={validatorAddr}
          onChangeText={setValidatorAddr}
        />
        <TextInput
          style={styles.input}
          placeholder="Amount (PAW)"
          placeholderTextColor="#666"
          value={stakeAmount}
          onChangeText={setStakeAmount}
          keyboardType="numeric"
        />
        <TouchableOpacity style={styles.button} onPress={handleDelegate}>
          <Text style={styles.buttonText}>Sign Delegation</Text>
        </TouchableOpacity>

        <Text style={styles.sectionLabel}>Governance Vote</Text>
        <TextInput
          style={styles.input}
          placeholder="Proposal ID"
          placeholderTextColor="#666"
          value={proposalId}
          onChangeText={setProposalId}
          keyboardType="numeric"
        />
        <TextInput
          style={styles.input}
          placeholder="Option (e.g., VOTE_OPTION_YES)"
          placeholderTextColor="#666"
          value={voteOption}
          onChangeText={setVoteOption}
        />
        <TouchableOpacity style={styles.button} onPress={handleVote}>
          <Text style={styles.buttonText}>Sign Vote</Text>
        </TouchableOpacity>

        <Text style={styles.sectionLabel}>IBC Transfer</Text>
        <TextInput
          style={styles.input}
          placeholder="Receiver address"
          placeholderTextColor="#666"
          value={ibcReceiver}
          onChangeText={setIbcReceiver}
        />
        <TextInput
          style={styles.input}
          placeholder="Channel (e.g., channel-0)"
          placeholderTextColor="#666"
          value={ibcChannel}
          onChangeText={setIbcChannel}
        />
        <TextInput
          style={styles.input}
          placeholder="Amount (PAW)"
          placeholderTextColor="#666"
          value={ibcAmount}
          onChangeText={setIbcAmount}
          keyboardType="numeric"
        />
        <TouchableOpacity style={styles.button} onPress={handleIbc}>
          <Text style={styles.buttonText}>Sign IBC Transfer</Text>
        </TouchableOpacity>
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
  helper: {
    color: '#9aa4b5',
    fontSize: 12,
    marginBottom: 10,
  },
  sectionLabel: {
    color: '#ccc',
    marginTop: 10,
    marginBottom: 6,
    fontWeight: '600',
    fontSize: 13,
  },
  input: {
    backgroundColor: '#15151d',
    borderRadius: 8,
    padding: 12,
    color: '#fff',
    marginBottom: 10,
    borderWidth: 1,
    borderColor: '#1f2a3d',
  },
  button: {
    backgroundColor: '#4A90E2',
    paddingVertical: 12,
    borderRadius: 8,
    alignItems: 'center',
    marginBottom: 12,
  },
  buttonText: {
    color: '#fff',
    fontWeight: '700',
  },
});

export default HomeScreen;
