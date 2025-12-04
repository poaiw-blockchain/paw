/**
 * Screen Component Tests
 */

import React from 'react';
import {render, fireEvent, waitFor} from '@testing-library/react-native';
import WelcomeScreen from '../src/screens/WelcomeScreen';
import HomeScreen from '../src/screens/HomeScreen';

// Mock navigation
const mockNavigation = {
  navigate: jest.fn(),
  goBack: jest.fn(),
  replace: jest.fn(),
};

// Mock services
jest.mock('../src/services/WalletService');
jest.mock('../src/services/PawAPI');
jest.mock('../src/services/KeyStore');

describe('Screen Components', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('WelcomeScreen', () => {
    it('should render welcome screen correctly', () => {
      const {getByText} = render(
        <WelcomeScreen navigation={mockNavigation} />,
      );

      expect(getByText('PAW Wallet')).toBeTruthy();
      expect(getByText('Create New Wallet')).toBeTruthy();
      expect(getByText('Import Wallet')).toBeTruthy();
    });

    it('should navigate to create wallet', () => {
      const {getByText} = render(
        <WelcomeScreen navigation={mockNavigation} />,
      );

      const createButton = getByText('Create New Wallet');
      fireEvent.press(createButton);

      expect(mockNavigation.navigate).toHaveBeenCalledWith('CreateWallet');
    });

    it('should navigate to import wallet', () => {
      const {getByText} = render(
        <WelcomeScreen navigation={mockNavigation} />,
      );

      const importButton = getByText('Import Wallet');
      fireEvent.press(importButton);

      expect(mockNavigation.navigate).toHaveBeenCalledWith('ImportWallet');
    });
  });

  describe('HomeScreen', () => {
    it('should show loading state initially', () => {
      const {getByText} = render(
        <HomeScreen navigation={mockNavigation} />,
      );

      expect(getByText('Loading wallet...')).toBeTruthy();
    });
  });
});
