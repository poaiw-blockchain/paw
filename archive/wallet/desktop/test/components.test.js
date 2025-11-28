/**
 * Component Tests
 * Tests for React components
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Wallet from '../src/components/Wallet';
import Send from '../src/components/Send';
import Receive from '../src/components/Receive';
import History from '../src/components/History';
import AddressBook from '../src/components/AddressBook';
import Settings from '../src/components/Settings';

describe('Component Tests', () => {
  describe('Wallet Component', () => {
    test('should render wallet balance', () => {
      const mockWalletData = {
        address: 'paw1test123',
        publicKey: '0x123456'
      };

      render(<Wallet walletData={mockWalletData} />);

      expect(screen.getByText(/balance/i)).toBeInTheDocument();
    });

    test('should display loading state', () => {
      const mockWalletData = {
        address: 'paw1test123'
      };

      render(<Wallet walletData={mockWalletData} />);

      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });
  });

  describe('Send Component', () => {
    test('should render send form', () => {
      const mockWalletData = {
        address: 'paw1sender'
      };

      render(<Send walletData={mockWalletData} />);

      expect(screen.getByPlaceholderText(/paw1.../i)).toBeInTheDocument();
      expect(screen.getByLabelText(/amount/i)).toBeInTheDocument();
    });

    test('should validate recipient address', async () => {
      const mockWalletData = {
        address: 'paw1sender'
      };

      render(<Send walletData={mockWalletData} />);

      const recipientInput = screen.getByPlaceholderText(/paw1.../i);
      const previewButton = screen.getByText(/preview/i);

      await userEvent.type(recipientInput, 'invalid-address');
      fireEvent.click(previewButton);

      await waitFor(() => {
        expect(screen.getByText(/invalid/i)).toBeInTheDocument();
      });
    });
  });

  describe('Receive Component', () => {
    test('should display wallet address', () => {
      const mockWalletData = {
        address: 'paw1receiver123'
      };

      render(<Receive walletData={mockWalletData} />);

      expect(screen.getByText(/paw1receiver123/i)).toBeInTheDocument();
    });

    test('should copy address to clipboard', async () => {
      const mockWalletData = {
        address: 'paw1receiver123'
      };

      render(<Receive walletData={mockWalletData} />);

      const copyButton = screen.getByText(/copy/i);
      fireEvent.click(copyButton);

      await waitFor(() => {
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith('paw1receiver123');
      });
    });
  });

  describe('History Component', () => {
    test('should render transaction list', () => {
      const mockWalletData = {
        address: 'paw1test'
      };

      render(<History walletData={mockWalletData} />);

      expect(screen.getByText(/transaction history/i)).toBeInTheDocument();
    });

    test('should display empty state', async () => {
      const mockWalletData = {
        address: 'paw1test'
      };

      render(<History walletData={mockWalletData} />);

      await waitFor(() => {
        expect(screen.getByText(/no transactions/i)).toBeInTheDocument();
      });
    });
  });

  describe('AddressBook Component', () => {
    test('should render address book', () => {
      render(<AddressBook />);

      expect(screen.getByText(/address book/i)).toBeInTheDocument();
    });

    test('should show add address form', () => {
      render(<AddressBook />);

      const addButton = screen.getByText(/add address/i);
      fireEvent.click(addButton);

      expect(screen.getByPlaceholderText(/alice's wallet/i)).toBeInTheDocument();
    });
  });

  describe('Settings Component', () => {
    test('should render settings', () => {
      render(<Settings />);

      expect(screen.getByText(/network settings/i)).toBeInTheDocument();
    });

    test('should save settings', async () => {
      render(<Settings />);

      const saveButton = screen.getByText(/save settings/i);
      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText(/saved/i)).toBeInTheDocument();
      });
    });
  });
});
