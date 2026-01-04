const form = document.getElementById('faucet-form');
const addressInput = document.getElementById('address');
const submitButton = document.getElementById('submit-btn');
const statusEl = document.getElementById('status');
const messageEl = document.getElementById('message');

const FAUCET_ENDPOINT = '/faucet';
const HEALTH_ENDPOINT = '/health';

function setMessage(text, isError = false) {
    if (!text) {
        messageEl.style.display = 'none';
        messageEl.textContent = '';
        messageEl.classList.remove('error');
        return;
    }
    messageEl.textContent = text;
    messageEl.style.display = 'block';
    messageEl.classList.toggle('error', isError);
}

async function checkStatus() {
    try {
        const response = await fetch(HEALTH_ENDPOINT, { method: 'GET' });
        const data = await response.json();
        if (response.ok && data.status === 'ok') {
            statusEl.textContent = 'Faucet online';
            statusEl.style.background = 'rgba(88, 178, 108, 0.18)';
            statusEl.style.color = '#7ee39a';
        } else {
            statusEl.textContent = 'Faucet reporting issues';
            statusEl.style.background = 'rgba(255, 107, 107, 0.18)';
            statusEl.style.color = '#ff9e9e';
        }
    } catch (error) {
        statusEl.textContent = 'Faucet unreachable';
        statusEl.style.background = 'rgba(255, 107, 107, 0.18)';
        statusEl.style.color = '#ff9e9e';
    }
}

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    setMessage('');

    if (!form.reportValidity()) {
        return;
    }

    submitButton.disabled = true;
    submitButton.textContent = 'Sending...';

    try {
        const response = await fetch(FAUCET_ENDPOINT, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address: addressInput.value.trim() }),
        });

        const data = await response.json();
        if (response.ok && data.success) {
            setMessage(`${data.message} Tx: ${data.txhash || 'pending'}`);
            form.reset();
        } else {
            setMessage(data.message || 'Request failed', true);
        }
    } catch (error) {
        setMessage('Network error. Please try again.', true);
    } finally {
        submitButton.disabled = false;
        submitButton.textContent = 'Send 100 PAW';
    }
});

checkStatus();
