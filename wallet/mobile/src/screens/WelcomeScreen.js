import React from 'react';
import {View, Text, StyleSheet, TouchableOpacity} from 'react-native';

const WelcomeScreen = ({navigation}) => {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>PAW Wallet</Text>
      <Text style={styles.subtitle}>Secure mobile wallet for the PAW chain</Text>

      <TouchableOpacity
        style={styles.primaryButton}
        onPress={() => navigation.navigate('CreateWallet')}>
        <Text style={styles.primaryButtonText}>Create New Wallet</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={styles.secondaryButton}
        onPress={() => navigation.navigate('ImportWallet')}>
        <Text style={styles.secondaryButtonText}>Import Wallet</Text>
      </TouchableOpacity>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#111',
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
  },
  title: {
    fontSize: 32,
    fontWeight: '700',
    color: '#fff',
  },
  subtitle: {
    fontSize: 16,
    color: '#aaa',
    marginTop: 8,
    marginBottom: 40,
    textAlign: 'center',
  },
  primaryButton: {
    width: '100%',
    backgroundColor: '#4A90E2',
    paddingVertical: 14,
    borderRadius: 8,
    alignItems: 'center',
    marginBottom: 16,
  },
  primaryButtonText: {
    color: '#fff',
    fontWeight: '600',
  },
  secondaryButton: {
    width: '100%',
    paddingVertical: 14,
    borderRadius: 8,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#4A90E2',
  },
  secondaryButtonText: {
    color: '#4A90E2',
    fontWeight: '600',
  },
});

export default WelcomeScreen;
