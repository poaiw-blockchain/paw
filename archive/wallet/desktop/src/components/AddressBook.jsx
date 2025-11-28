import React, { useState, useEffect } from 'react';

const AddressBook = () => {
  const [addresses, setAddresses] = useState([]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const [editingIndex, setEditingIndex] = useState(null);

  useEffect(() => {
    loadAddresses();
  }, []);

  const loadAddresses = async () => {
    try {
      if (window.electron?.store) {
        const saved = await window.electron.store.get('addressBook');
        if (saved) {
          setAddresses(saved);
        }
      }
    } catch (err) {
      console.error('Failed to load addresses:', err);
    }
  };

  const saveAddresses = async (newAddresses) => {
    try {
      if (window.electron?.store) {
        await window.electron.store.set('addressBook', newAddresses);
      }
      setAddresses(newAddresses);
    } catch (err) {
      console.error('Failed to save addresses:', err);
      setError('Failed to save address');
    }
  };

  const handleAdd = () => {
    try {
      setError('');

      if (!name.trim()) {
        throw new Error('Name is required');
      }

      if (!address.trim()) {
        throw new Error('Address is required');
      }

      if (!address.startsWith('paw')) {
        throw new Error('Invalid PAW address');
      }

      const newAddress = {
        name: name.trim(),
        address: address.trim(),
        note: note.trim(),
        createdAt: new Date().toISOString()
      };

      let newAddresses;
      if (editingIndex !== null) {
        newAddresses = [...addresses];
        newAddresses[editingIndex] = { ...newAddress, createdAt: addresses[editingIndex].createdAt };
        setEditingIndex(null);
      } else {
        newAddresses = [...addresses, newAddress];
      }

      saveAddresses(newAddresses);
      setName('');
      setAddress('');
      setNote('');
      setShowAddForm(false);
    } catch (err) {
      setError(err.message);
    }
  };

  const handleEdit = (index) => {
    const addr = addresses[index];
    setName(addr.name);
    setAddress(addr.address);
    setNote(addr.note || '');
    setEditingIndex(index);
    setShowAddForm(true);
  };

  const handleDelete = async (index) => {
    if (window.electron?.dialog) {
      const result = await window.electron.dialog.showMessageBox({
        type: 'question',
        title: 'Delete Address',
        message: 'Are you sure you want to delete this address?',
        buttons: ['Cancel', 'Delete'],
        defaultId: 0,
        cancelId: 0
      });

      if (result.response === 1) {
        const newAddresses = addresses.filter((_, i) => i !== index);
        saveAddresses(newAddresses);
      }
    } else {
      if (window.confirm('Are you sure you want to delete this address?')) {
        const newAddresses = addresses.filter((_, i) => i !== index);
        saveAddresses(newAddresses);
      }
    }
  };

  const handleCopy = async (addr) => {
    try {
      await navigator.clipboard.writeText(addr);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const handleCancel = () => {
    setName('');
    setAddress('');
    setNote('');
    setError('');
    setEditingIndex(null);
    setShowAddForm(false);
  };

  return (
    <div className="content">
      <div className="card">
        <div className="flex-between mb-20">
          <h3 className="card-header" style={{ marginBottom: 0 }}>Address Book</h3>
          <button
            className="btn btn-primary"
            onClick={() => setShowAddForm(!showAddForm)}
          >
            {showAddForm ? 'Cancel' : 'Add Address'}
          </button>
        </div>

        {showAddForm && (
          <div style={{
            background: 'var(--bg-primary)',
            padding: '20px',
            borderRadius: '6px',
            marginBottom: '20px'
          }}>
            <h4 style={{ marginBottom: '15px' }}>
              {editingIndex !== null ? 'Edit Address' : 'Add New Address'}
            </h4>
            <div className="form-group">
              <label className="form-label">Name</label>
              <input
                type="text"
                className="form-input"
                placeholder="e.g., Alice's Wallet"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Address</label>
              <input
                type="text"
                className="form-input"
                placeholder="paw1..."
                value={address}
                onChange={(e) => setAddress(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Note (Optional)</label>
              <input
                type="text"
                className="form-input"
                placeholder="Add a note"
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
            </div>
            {error && <div className="text-error mb-20">{error}</div>}
            <div className="flex gap-10">
              <button className="btn btn-secondary" onClick={handleCancel}>
                Cancel
              </button>
              <button className="btn btn-primary" onClick={handleAdd}>
                {editingIndex !== null ? 'Update' : 'Add'}
              </button>
            </div>
          </div>
        )}

        {addresses.length > 0 ? (
          <div style={{ overflowX: 'auto' }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Address</th>
                  <th>Note</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {addresses.map((addr, index) => (
                  <tr key={index}>
                    <td style={{ fontWeight: '600' }}>{addr.name}</td>
                    <td>
                      <span
                        style={{
                          fontFamily: 'monospace',
                          fontSize: '12px',
                          cursor: 'pointer'
                        }}
                        onClick={() => handleCopy(addr.address)}
                        title="Click to copy"
                      >
                        {addr.address.substring(0, 20)}...
                      </span>
                    </td>
                    <td className="text-muted">{addr.note || '-'}</td>
                    <td>
                      <div className="flex gap-10">
                        <button
                          className="btn btn-secondary"
                          style={{ padding: '5px 10px', fontSize: '12px' }}
                          onClick={() => handleEdit(index)}
                        >
                          Edit
                        </button>
                        <button
                          className="btn btn-danger"
                          style={{ padding: '5px 10px', fontSize: '12px' }}
                          onClick={() => handleDelete(index)}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center text-muted">
            <p>No addresses saved</p>
            <p style={{ fontSize: '12px', marginTop: '10px' }}>
              Add frequently used addresses for quick access
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default AddressBook;
