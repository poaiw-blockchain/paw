import React, { useState, useEffect } from 'react';
import Wallet from './components/Wallet';
import Send from './components/Send';
import Receive from './components/Receive';
import History from './components/History';
import AddressBook from './components/AddressBook';
import Settings from './components/Settings';
import Setup from './components/Setup';
import { KeystoreService } from './services/keystore';

const App = () => {
  const [currentView, setCurrentView] = useState('wallet');
  const [walletData, setWalletData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    initializeWallet();
    setupMenuListener();

    return () => {
      if (window.electron?.removeMenuActionListener) {
        window.electron.removeMenuActionListener();
      }
    };
  }, []);

  const initializeWallet = async () => {
    try {
      setLoading(true);
      const keystoreService = new KeystoreService();
      const wallet = await keystoreService.getWallet();
      setWalletData(wallet);
    } catch (err) {
      console.error('Failed to initialize wallet:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const setupMenuListener = () => {
    if (window.electron?.onMenuAction) {
      window.electron.onMenuAction((action) => {
        switch (action) {
          case 'new-wallet':
            handleNewWallet();
            break;
          case 'import-wallet':
            handleImportWallet();
            break;
          case 'backup-wallet':
            handleBackupWallet();
            break;
          case 'send':
            setCurrentView('send');
            break;
          case 'receive':
            setCurrentView('receive');
            break;
          case 'history':
            setCurrentView('history');
            break;
          case 'address-book':
            setCurrentView('addressBook');
            break;
          case 'settings':
            setCurrentView('settings');
            break;
          default:
            break;
        }
      });
    }
  };

  const handleNewWallet = async () => {
    if (window.electron?.dialog) {
      const result = await window.electron.dialog.showMessageBox({
        type: 'warning',
        title: 'Create New Wallet',
        message: 'This will replace your current wallet. Make sure you have backed up your mnemonic phrase.',
        buttons: ['Cancel', 'Create New Wallet'],
        defaultId: 0,
        cancelId: 0
      });

      if (result.response === 1) {
        setCurrentView('setup');
      }
    }
  };

  const handleImportWallet = () => {
    setCurrentView('setup');
  };

  const handleBackupWallet = async () => {
    try {
      const keystoreService = new KeystoreService();
      const mnemonic = await keystoreService.getMnemonic();

      if (mnemonic && window.electron?.dialog) {
        await window.electron.dialog.showMessageBox({
          type: 'info',
          title: 'Backup Mnemonic',
          message: 'Write down these 24 words in order and store them safely:',
          detail: mnemonic,
          buttons: ['I have written it down']
        });
      }
    } catch (err) {
      console.error('Failed to backup wallet:', err);
      if (window.electron?.dialog) {
        await window.electron.dialog.showMessageBox({
          type: 'error',
          title: 'Backup Failed',
          message: err.message
        });
      }
    }
  };

  const handleWalletCreated = (newWallet) => {
    setWalletData(newWallet);
    setCurrentView('wallet');
  };

  const renderContent = () => {
    if (loading) {
      return (
        <div className="content text-center">
          <div className="loading-spinner"></div>
          <p className="text-muted">Loading wallet...</p>
        </div>
      );
    }

    if (!walletData && currentView !== 'setup') {
      return <Setup onWalletCreated={handleWalletCreated} />;
    }

    switch (currentView) {
      case 'wallet':
        return <Wallet walletData={walletData} onRefresh={initializeWallet} />;
      case 'send':
        return <Send walletData={walletData} onSuccess={initializeWallet} />;
      case 'receive':
        return <Receive walletData={walletData} />;
      case 'history':
        return <History walletData={walletData} />;
      case 'addressBook':
        return <AddressBook />;
      case 'settings':
        return <Settings onWalletReset={() => {
          setWalletData(null);
          setCurrentView('setup');
        }} />;
      case 'setup':
        return <Setup onWalletCreated={handleWalletCreated} />;
      default:
        return <Wallet walletData={walletData} onRefresh={initializeWallet} />;
    }
  };

  if (error) {
    return (
      <div className="app">
        <div className="main-content">
          <div className="content text-center">
            <h2 className="text-error">Error</h2>
            <p className="text-muted mt-20">{error}</p>
            <button className="btn btn-primary mt-20" onClick={initializeWallet}>
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="app">
      {walletData && (
        <aside className="sidebar">
          <div className="sidebar-header">
            <h1>PAW</h1>
          </div>
          <nav className="nav">
            <div
              className={`nav-item ${currentView === 'wallet' ? 'active' : ''}`}
              onClick={() => setCurrentView('wallet')}
            >
              Wallet
            </div>
            <div
              className={`nav-item ${currentView === 'send' ? 'active' : ''}`}
              onClick={() => setCurrentView('send')}
            >
              Send
            </div>
            <div
              className={`nav-item ${currentView === 'receive' ? 'active' : ''}`}
              onClick={() => setCurrentView('receive')}
            >
              Receive
            </div>
            <div
              className={`nav-item ${currentView === 'history' ? 'active' : ''}`}
              onClick={() => setCurrentView('history')}
            >
              History
            </div>
            <div
              className={`nav-item ${currentView === 'addressBook' ? 'active' : ''}`}
              onClick={() => setCurrentView('addressBook')}
            >
              Address Book
            </div>
            <div
              className={`nav-item ${currentView === 'settings' ? 'active' : ''}`}
              onClick={() => setCurrentView('settings')}
            >
              Settings
            </div>
          </nav>
        </aside>
      )}
      <main className="main-content">
        {walletData && (
          <header className="header">
            <div>
              <h2>{currentView.charAt(0).toUpperCase() + currentView.slice(1)}</h2>
              <div className="wallet-address">{walletData.address}</div>
            </div>
          </header>
        )}
        {renderContent()}
      </main>
    </div>
  );
};

export default App;
