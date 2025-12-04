import React, { useState } from 'react';

const Receive = ({ walletData }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(walletData.address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const generateQRCode = () => {
    // Simple QR code representation (in production, use a proper QR code library)
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${walletData.address}`;
  };

  return (
    <div className="content">
      <div className="card" style={{ maxWidth: '600px', margin: '0 auto' }}>
        <h3 className="card-header">Receive PAW</h3>

        <div className="text-center mb-20">
          <p className="text-muted mb-20">
            Share this address to receive PAW tokens
          </p>

          <div style={{
            background: 'var(--bg-primary)',
            padding: '20px',
            borderRadius: '8px',
            marginBottom: '20px'
          }}>
            <img
              src={generateQRCode()}
              alt="Wallet QR Code"
              style={{
                maxWidth: '200px',
                height: 'auto',
                border: '2px solid var(--border)',
                borderRadius: '8px'
              }}
              onError={(e) => {
                e.target.style.display = 'none';
              }}
            />
          </div>

          <div style={{
            background: 'var(--bg-primary)',
            padding: '15px',
            borderRadius: '6px',
            marginBottom: '15px',
            fontFamily: 'monospace',
            fontSize: '13px',
            wordBreak: 'break-all',
            userSelect: 'all'
          }}>
            {walletData.address}
          </div>

          <button
            className="btn btn-primary"
            onClick={handleCopy}
            style={{ minWidth: '150px' }}
          >
            {copied ? 'Copied!' : 'Copy Address'}
          </button>
        </div>

        <div style={{
          background: 'rgba(122, 162, 247, 0.1)',
          border: '1px solid var(--accent)',
          borderRadius: '6px',
          padding: '15px',
          marginTop: '20px'
        }}>
          <div style={{ fontSize: '14px', fontWeight: '600', marginBottom: '10px' }}>
            Important Notes
          </div>
          <ul style={{ fontSize: '12px', lineHeight: '1.6', paddingLeft: '20px' }}>
            <li>Only send PAW tokens to this address</li>
            <li>Sending other tokens may result in permanent loss</li>
            <li>Always verify the address before sharing</li>
            <li>Consider using a memo for exchanges and services</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default Receive;
