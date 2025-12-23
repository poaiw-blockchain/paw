// Package keeper implements the Oracle module keeper for decentralized price feeds.
//
// The Oracle module aggregates price data from validator submissions using
// weighted median calculation, time-weighted average pricing (TWAP), and
// geographic diversity enforcement. Provides tamper-resistant price feeds for
// on-chain protocols with cryptoeconomic security guarantees.
//
// # Core Functionality
//
// Price Aggregation: Validators submit price observations with timestamps and
// signatures. The keeper aggregates submissions using voting-power-weighted
// median to resist manipulation and produce canonical prices.
//
// TWAP Calculation: Maintains time-weighted average prices over configurable
// windows (1h, 4h, 24h) for smoothing volatility and preventing oracle attacks.
//
// Geographic Diversity: Validates validator geographic distribution using GeoIP
// lookups to prevent regional manipulation. Enforces minimum continent/country
// diversity thresholds for price acceptance.
//
// Validator Incentives: Rewards accurate price submissions and slashes validators
// for excessive deviations, late submissions, or inactivity. Tracks per-validator
// statistics and reputation scores.
//
// IBC Price Feeds: Cross-chain price data sharing via IBC packets. Remote chains
// can query canonical prices or subscribe to price update feeds.
//
// Security Features: Circuit breakers for abnormal price movements, submission
// rate limiting, replay attack prevention, and comprehensive metrics.
//
// # Key Types
//
// Keeper: Main module keeper managing price state, validator submissions, and
// geographic verification via GeoIPManager.
//
// Price: Aggregated price for an asset with value, timestamp, block height,
// and validator participation count.
//
// PriceSubmission: Individual validator observation with asset, price, timestamp,
// and validator signature.
//
// GeoIPManager: Geographic IP address validation using MaxMind GeoLite2 database
// for enforcing validator location diversity.
//
// # Usage Patterns
//
// Submitting a price:
//
//	err := keeper.SubmitPrice(ctx, validator, asset, price, timestamp)
//
// Querying aggregated price:
//
//	price, err := keeper.GetPrice(ctx, asset)
//
// Calculating TWAP:
//
//	twap, err := keeper.CalculateTWAP(ctx, asset, startTime, endTime)
//
// # IBC Port
//
// Binds to the "oracle" IBC port for cross-chain price feed distribution.
// Authorized channels relay price updates to connected blockchains.
//
// # Metrics
//
// Exposes Prometheus metrics for price updates, validator participation,
// deviation rates, and aggregation latency via OracleMetrics.
package keeper
