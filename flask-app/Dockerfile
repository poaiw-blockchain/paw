FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

ENV FLASK_APP=app.py
ENV RPC_URL=http://host.docker.internal:26657
ENV CHAIN_ID=paw-testnet-1

EXPOSE 11080

# Use gunicorn for production
CMD ["gunicorn", "--bind", "0.0.0.0:11080", "--workers", "2", "--timeout", "120", "app:app"]
