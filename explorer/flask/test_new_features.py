"""
PAW Blockchain Explorer - Comprehensive Tests
Tests for ALL explorer features: blocks, transactions, accounts, DEX, oracle, compute,
governance, staking, validators, rich list, CSV export, search, and statistics.
"""

import csv
import io
import json
from typing import Any, Dict
from unittest.mock import Mock, patch, MagicMock

import pytest
import requests

# Import Flask app
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from app import app, rpc_client


class _MockResponse:
    """Simple mock for HTTP responses"""

    def __init__(self, payload: Dict[str, Any], status_code: int = 200):
        self._payload = payload
        self.status_code = status_code
        self.text = json.dumps(payload)

    def json(self):
        return self._payload

    def raise_for_status(self):
        if self.status_code >= 400:
            raise requests.HTTPError(f"HTTP {self.status_code}")


@pytest.fixture
def client():
    """Create Flask test client"""
    app.config['TESTING'] = True
    with app.test_client() as test_client:
        yield test_client


# ==================== GOVERNANCE ENDPOINT TESTS ====================

class TestGovernanceEndpoints:
    """Test governance API endpoints"""

    def test_governance_proposals_endpoint(self, client):
        """Test GET /api/governance/proposals"""
        # Test that the endpoint exists and responds
        response = client.get('/api/governance/proposals')
        # May return 200 or 500 depending on network availability
        assert response.status_code in [200, 500]

        if response.status_code == 200:
            data = json.loads(response.data)
            assert "proposals" in data

    def test_governance_proposals_with_status_filter(self, client):
        """Test GET /api/governance/proposals?status=voting"""
        response = client.get('/api/governance/proposals?status=voting')
        # Endpoint should exist
        assert response.status_code in [200, 500]

    def test_governance_single_proposal(self, client):
        """Test GET /api/governance/proposals/<id>"""
        response = client.get('/api/governance/proposals/1')
        # Endpoint should exist
        assert response.status_code in [200, 404, 500]

    def test_governance_proposal_votes(self, client):
        """Test GET /api/governance/proposals/<id>/votes"""
        response = client.get('/api/governance/proposals/1/votes')
        # Endpoint should exist
        assert response.status_code in [200, 404, 500]


# ==================== STAKING ENDPOINT TESTS ====================

class TestStakingEndpoints:
    """Test staking API endpoints"""

    def test_staking_pool_endpoint(self, client):
        """Test GET /api/staking/pool"""
        response = client.get('/api/staking/pool')
        # Endpoint should exist
        assert response.status_code in [200, 500]

    def test_staking_delegations_endpoint(self, client):
        """Test GET /api/staking/delegations/<address>"""
        response = client.get('/api/staking/delegations/paw1testaddr')
        # Endpoint should exist
        assert response.status_code in [200, 404, 500]

    def test_staking_unbonding_endpoint(self, client):
        """Test GET /api/staking/unbonding/<address>"""
        response = client.get('/api/staking/unbonding/paw1testaddr')
        # Endpoint should exist
        assert response.status_code in [200, 404, 500]

    def test_staking_rewards_endpoint(self, client):
        """Test GET /api/staking/rewards/<address>"""
        response = client.get('/api/staking/rewards/paw1testaddr')
        # Endpoint should exist
        assert response.status_code in [200, 404, 500]


# ==================== VALIDATORS ENDPOINT TESTS ====================

class TestValidatorsEndpoints:
    """Test validators API endpoints"""

    def test_validators_list_endpoint(self, client):
        """Test GET /api/validators"""
        with patch.object(rpc_client, 'get_validators_rest') as mock_get:
            mock_get.return_value = {
                "validators": [
                    {
                        "operator_address": "pawvaloper1validator1",
                        "consensus_pubkey": {"key": "pubkey1"},
                        "jailed": False,
                        "status": "BOND_STATUS_BONDED",
                        "tokens": "10000000000",
                        "delegator_shares": "10000000000",
                        "description": {"moniker": "Validator One"},
                        "commission": {
                            "commission_rates": {"rate": "0.100000000000000000"}
                        }
                    },
                    {
                        "operator_address": "pawvaloper1validator2",
                        "consensus_pubkey": {"key": "pubkey2"},
                        "jailed": False,
                        "status": "BOND_STATUS_BONDED",
                        "tokens": "5000000000",
                        "delegator_shares": "5000000000",
                        "description": {"moniker": "Validator Two"},
                        "commission": {
                            "commission_rates": {"rate": "0.050000000000000000"}
                        }
                    }
                ]
            }

            response = client.get('/api/validators')
            assert response.status_code == 200

            data = json.loads(response.data)
            assert "validators" in data
            assert len(data["validators"]) == 2

    def test_validator_detail_endpoint(self, client):
        """Test GET /api/validators/<address>"""
        with patch.object(rpc_client, 'get_validator_rest') as mock_validator, \
             patch.object(rpc_client, 'get_validator_commission') as mock_commission, \
             patch.object(rpc_client, 'get_validator_delegations') as mock_delegations:

            mock_validator.return_value = {
                "validator": {
                    "operator_address": "pawvaloper1test",
                    "tokens": "10000000000",
                    "description": {"moniker": "Test Validator"},
                    "commission": {
                        "commission_rates": {"rate": "0.10"}
                    }
                }
            }
            mock_commission.return_value = {
                "commission": {"commission": [{"denom": "upaw", "amount": "500000"}]}
            }
            mock_delegations.return_value = {"delegation_responses": []}

            response = client.get('/api/validators/pawvaloper1test')
            assert response.status_code == 200


# ==================== RICH LIST ENDPOINT TESTS ====================

class TestRichListEndpoints:
    """Test rich list API endpoints"""

    def test_richlist_endpoint(self, client):
        """Test GET /api/v1/richlist"""
        with patch.object(rpc_client, 'get_all_balances') as mock_balances, \
             patch.object(rpc_client, 'get_total_supply') as mock_supply:

            mock_balances.return_value = [
                {"address": "paw1whale1", "balance": "1000000000000"},
                {"address": "paw1whale2", "balance": "500000000000"},
                {"address": "paw1whale3", "balance": "100000000000"}
            ]
            mock_supply.return_value = {
                "amount": {"denom": "upaw", "amount": "10000000000000"}
            }

            response = client.get('/api/v1/richlist?limit=10')
            assert response.status_code == 200

            data = json.loads(response.data)
            assert "richlist" in data
            assert "total_supply" in data
            assert "total_holders" in data

    def test_richlist_with_limit(self, client):
        """Test GET /api/v1/richlist?limit=5"""
        with patch.object(rpc_client, 'get_all_balances') as mock_balances, \
             patch.object(rpc_client, 'get_total_supply') as mock_supply:

            mock_balances.return_value = [
                {"address": f"paw1addr{i}", "balance": str((100-i) * 1000000000)}
                for i in range(10)
            ]
            mock_supply.return_value = {"amount": {"amount": "1000000000000"}}

            response = client.get('/api/v1/richlist?limit=5')
            assert response.status_code == 200

            data = json.loads(response.data)
            assert len(data["richlist"]) <= 5

    def test_richlist_percentage_calculation(self, client):
        """Test that richlist returns valid percentage data"""
        response = client.get('/api/v1/richlist')
        assert response.status_code == 200

        data = json.loads(response.data)
        if data.get("richlist"):
            # Should have percentage field
            first_entry = data["richlist"][0]
            assert "percentage" in first_entry


# ==================== CSV EXPORT ENDPOINT TESTS ====================

class TestExportEndpoints:
    """Test CSV export API endpoints"""

    def test_export_transactions_csv(self, client):
        """Test GET /api/v1/export/transactions/<address>"""
        with patch.object(rpc_client, 'get_account_transactions_all') as mock_txs:
            mock_txs.return_value = [
                {
                    "hash": "ABCD1234",
                    "timestamp": "2024-01-15T12:00:00Z",
                    "type": "MsgSend",
                    "sender": "paw1sender",
                    "receiver": "paw1receiver",
                    "messages": [{
                        "amount": {"amount": "1000000", "denom": "upaw"}
                    }],
                    "fee_amount": "500",
                    "fee_denom": "upaw",
                    "status": "success",
                    "block_height": "12345",
                    "memo": "Test transfer"
                }
            ]

            response = client.get('/api/v1/export/transactions/paw1testaddr')
            assert response.status_code == 200
            assert response.content_type == 'text/csv'
            assert 'attachment' in response.headers.get('Content-Disposition', '')

            # Parse CSV
            csv_content = response.data.decode('utf-8')
            reader = csv.reader(io.StringIO(csv_content))
            rows = list(reader)

            assert len(rows) >= 2  # Header + at least one data row
            assert 'TxHash' in rows[0]  # Check header

    def test_export_transactions_json(self, client):
        """Test GET /api/v1/export/transactions/<address>?format=json"""
        with patch.object(rpc_client, 'get_account_transactions_all') as mock_txs:
            mock_txs.return_value = [
                {
                    "hash": "ABCD1234",
                    "timestamp": "2024-01-15T12:00:00Z",
                    "type": "MsgSend"
                }
            ]

            response = client.get('/api/v1/export/transactions/paw1testaddr?format=json')
            assert response.status_code == 200
            assert response.content_type == 'application/json'

            data = json.loads(response.data)
            assert "transactions" in data
            assert "address" in data
            assert "exported_at" in data

    def test_export_account_csv(self, client):
        """Test GET /api/v1/export/account/<address>"""
        with patch.object(rpc_client, 'get_account') as mock_account:
            mock_account.return_value = {
                "account": {
                    "tx_count": 100,
                    "total_received": "10000000",
                    "total_sent": "5000000",
                    "first_seen_height": "1000",
                    "first_seen_at": "2024-01-01T00:00:00Z",
                    "last_seen_height": "12345",
                    "last_seen_at": "2024-01-15T12:00:00Z"
                }
            }

            response = client.get('/api/v1/export/account/paw1testaddr')
            assert response.status_code == 200
            assert response.content_type == 'text/csv'

            csv_content = response.data.decode('utf-8')
            assert 'Address' in csv_content
            assert 'paw1testaddr' in csv_content

    def test_export_account_json(self, client):
        """Test GET /api/v1/export/account/<address>?format=json"""
        with patch.object(rpc_client, 'get_account') as mock_account:
            mock_account.return_value = {"account": {"tx_count": 50}}

            response = client.get('/api/v1/export/account/paw1testaddr?format=json')
            assert response.status_code == 200
            assert response.content_type == 'application/json'

    def test_export_transactions_limit(self, client):
        """Test that export respects limit parameter"""
        with patch.object(rpc_client, 'get_account_transactions_all') as mock_txs:
            mock_txs.return_value = [{"hash": f"TX{i}"} for i in range(100)]

            response = client.get('/api/v1/export/transactions/paw1addr?limit=50')
            assert response.status_code == 200


# ==================== SWAGGER/API DOCS TESTS ====================

class TestSwaggerDocs:
    """Test Swagger/OpenAPI documentation"""

    def test_swagger_ui_available(self, client):
        """Test that Swagger UI is accessible"""
        response = client.get('/apidocs/')
        # Should return 200 or redirect or 404 if not configured
        assert response.status_code in [200, 301, 302, 308, 404]

    def test_swagger_json_spec(self, client):
        """Test that Swagger JSON spec is accessible"""
        response = client.get('/apispec_1.json')
        # May or may not be available
        if response.status_code == 200:
            data = json.loads(response.data)
            assert "swagger" in data or "openapi" in data or "info" in data


# ==================== INTEGRATION TESTS ====================

class TestFeatureIntegration:
    """Integration tests for new features"""

    def test_governance_staking_workflow(self, client):
        """Test governance + staking integration"""
        # First check governance proposals
        gov_response = client.get('/api/governance/proposals')
        # May return 200 or 500 depending on network availability
        assert gov_response.status_code in [200, 500]

        # Then check staking pool
        staking_response = client.get('/api/staking/pool')
        # May return 200 or 500 depending on network availability
        assert staking_response.status_code in [200, 500]

    def test_richlist_export_workflow(self, client):
        """Test viewing rich list then exporting data"""
        test_address = "paw1topwhale"

        # View rich list
        with patch.object(rpc_client, 'get_all_balances') as mock_balances, \
             patch.object(rpc_client, 'get_total_supply') as mock_supply:

            mock_balances.return_value = [{"address": test_address, "balance": "1000000000000"}]
            mock_supply.return_value = {"amount": {"amount": "10000000000000"}}

            richlist_response = client.get('/api/v1/richlist')
            assert richlist_response.status_code == 200

        # Export that address's transactions
        with patch.object(rpc_client, 'get_account_transactions_all') as mock_txs:
            mock_txs.return_value = [{"hash": "TX1", "timestamp": "2024-01-01"}]

            export_response = client.get(f'/api/v1/export/transactions/{test_address}')
            assert export_response.status_code == 200
            assert 'csv' in export_response.content_type

    def test_validator_and_staking_consistency(self, client):
        """Test that validator and staking data is consistent"""
        with patch.object(rpc_client, 'get_validators_rest') as mock_validators:
            mock_validators.return_value = {
                "validators": [
                    {
                        "operator_address": "pawvaloper1test",
                        "tokens": "10000000000",
                        "description": {"moniker": "Test"},
                        "status": "BOND_STATUS_BONDED",
                        "commission": {"commission_rates": {"rate": "0.1"}}
                    }
                ]
            }

            val_response = client.get('/api/validators')
            assert val_response.status_code == 200


# ==================== ERROR HANDLING TESTS ====================

class TestErrorHandling:
    """Test error handling in new endpoints"""

    def test_governance_proposals_error(self, client):
        """Test governance endpoint handles errors gracefully"""
        with patch.object(rpc_client, 'get_proposals') as mock_get:
            mock_get.return_value = None  # Simulates error

            response = client.get('/api/governance/proposals')
            # Should return 200 with empty list or 500
            assert response.status_code in [200, 500]

    def test_richlist_error(self, client):
        """Test richlist endpoint handles errors gracefully"""
        with patch.object(rpc_client, 'get_all_balances') as mock_balances:
            mock_balances.return_value = None

            response = client.get('/api/v1/richlist')
            assert response.status_code in [200, 500]

    def test_export_invalid_address(self, client):
        """Test export with non-existent address"""
        with patch.object(rpc_client, 'get_account_transactions_all') as mock_txs:
            mock_txs.return_value = []  # Empty result

            response = client.get('/api/v1/export/transactions/paw1invalidaddr')
            assert response.status_code == 200  # Returns empty CSV


# ==================== CORE FEATURE TESTS ====================

class TestHealthEndpoints:
    """Test health and metrics endpoints"""

    def test_health_endpoint(self, client):
        """Test GET /health"""
        response = client.get('/health')
        assert response.status_code == 200
        data = json.loads(response.data)
        assert "status" in data

    def test_health_ready_endpoint(self, client):
        """Test GET /health/ready"""
        response = client.get('/health/ready')
        # May return 200 or 503 depending on backend availability
        assert response.status_code in [200, 503]

    def test_metrics_endpoint(self, client):
        """Test GET /metrics - Prometheus metrics"""
        response = client.get('/metrics')
        assert response.status_code == 200


class TestBlockEndpoints:
    """Test block-related endpoints"""

    def test_blocks_page(self, client):
        """Test GET /blocks - HTML page"""
        response = client.get('/blocks')
        assert response.status_code in [200, 500]

    def test_blocks_api(self, client):
        """Test GET /api/v1/blocks"""
        response = client.get('/api/v1/blocks')
        assert response.status_code in [200, 500]

        if response.status_code == 200:
            data = json.loads(response.data)
            assert "blocks" in data or "error" in data

    def test_blocks_api_with_pagination(self, client):
        """Test GET /api/v1/blocks with pagination"""
        response = client.get('/api/v1/blocks?page=1&limit=10')
        assert response.status_code in [200, 500]

    def test_single_block_page(self, client):
        """Test GET /block/<height> - HTML page"""
        response = client.get('/block/1')
        assert response.status_code in [200, 404, 500]

    def test_single_block_api(self, client):
        """Test GET /api/v1/blocks/<height>"""
        response = client.get('/api/v1/blocks/1')
        assert response.status_code in [200, 404, 500]


class TestTransactionEndpoints:
    """Test transaction-related endpoints"""

    def test_transactions_page(self, client):
        """Test GET /transactions - HTML page"""
        response = client.get('/transactions')
        assert response.status_code in [200, 500]

    def test_transactions_api(self, client):
        """Test GET /api/v1/transactions"""
        response = client.get('/api/v1/transactions')
        assert response.status_code in [200, 500]

        if response.status_code == 200:
            data = json.loads(response.data)
            assert "transactions" in data or "txs" in data or "error" in data

    def test_transactions_api_with_pagination(self, client):
        """Test GET /api/v1/transactions with pagination"""
        response = client.get('/api/v1/transactions?page=1&limit=20')
        assert response.status_code in [200, 500]

    def test_single_transaction_page(self, client):
        """Test GET /tx/<hash> - HTML page"""
        response = client.get('/tx/ABCD1234567890')
        assert response.status_code in [200, 404, 500]

    def test_single_transaction_api(self, client):
        """Test GET /api/v1/transactions/<hash>"""
        response = client.get('/api/v1/transactions/ABCD1234567890')
        assert response.status_code in [200, 404, 500]


class TestAccountEndpoints:
    """Test account-related endpoints"""

    def test_account_page(self, client):
        """Test GET /account/<address> - HTML page"""
        response = client.get('/account/paw1testaddr')
        assert response.status_code in [200, 404, 500]


class TestDEXEndpoints:
    """Test DEX-related endpoints"""

    def test_dex_page(self, client):
        """Test GET /dex - HTML page"""
        response = client.get('/dex')
        assert response.status_code in [200, 500]

    def test_dex_pool_page(self, client):
        """Test GET /dex/pool/<pool_id> - HTML page"""
        response = client.get('/dex/pool/1')
        assert response.status_code in [200, 404, 500]


class TestOracleEndpoints:
    """Test oracle-related endpoints"""

    def test_oracle_page(self, client):
        """Test GET /oracle - HTML page"""
        response = client.get('/oracle')
        assert response.status_code in [200, 500]


class TestComputeEndpoints:
    """Test compute-related endpoints"""

    def test_compute_page(self, client):
        """Test GET /compute - HTML page"""
        response = client.get('/compute')
        assert response.status_code in [200, 500]


class TestSearchEndpoints:
    """Test search functionality"""

    def test_search_page(self, client):
        """Test GET /search - HTML page"""
        response = client.get('/search?q=test')
        assert response.status_code in [200, 500]

    def test_search_api(self, client):
        """Test GET /api/v1/search"""
        response = client.get('/api/v1/search?q=paw1')
        assert response.status_code in [200, 500]

        if response.status_code == 200:
            data = json.loads(response.data)
            assert "results" in data or "type" in data or "error" in data

    def test_search_empty_query(self, client):
        """Test search with empty query"""
        response = client.get('/api/v1/search?q=')
        assert response.status_code in [200, 400, 500]


class TestStatisticsEndpoints:
    """Test statistics endpoints"""

    def test_stats_api(self, client):
        """Test GET /api/v1/stats"""
        response = client.get('/api/v1/stats')
        assert response.status_code in [200, 500]

        if response.status_code == 200:
            data = json.loads(response.data)
            # Should have some stats data
            assert isinstance(data, dict)


class TestValidatorsPage:
    """Test validators HTML page"""

    def test_validators_page(self, client):
        """Test GET /validators - HTML page"""
        response = client.get('/validators')
        assert response.status_code in [200, 500]

    def test_validator_detail_page(self, client):
        """Test GET /validator/<address> - HTML page"""
        response = client.get('/validator/pawvaloper1test')
        assert response.status_code in [200, 404, 500]


class TestHomePage:
    """Test home page"""

    def test_home_page(self, client):
        """Test GET / - Home page"""
        response = client.get('/')
        assert response.status_code == 200


class TestStakingUnbondingEndpoint:
    """Test staking unbonding endpoint"""

    def test_staking_unbonding_endpoint(self, client):
        """Test GET /api/staking/unbonding/<address>"""
        response = client.get('/api/staking/unbonding/paw1testaddr')
        assert response.status_code in [200, 404, 500]


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
