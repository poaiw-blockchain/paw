package api

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gin-gonic/gin"
)

// Validation constants
const (
	MaxRequestSize    = 1 << 20 // 1 MB
	MaxUsernameLength = 50
	MinUsernameLength = 3
	MinPasswordLength = 8
	MaxPasswordLength = 128
	MaxMemoLength     = 256
	MaxAmountLength   = 30
	MaxAddressLength  = 100
	MaxOrderIDLength  = 64
	MaxSwapIDLength   = 64
	MaxPoolIDLength   = 64
	MaxHashLength     = 128
	MaxSecretLength   = 128
	MaxTxHashLength   = 128
	MaxPathLength     = 256
)

// Regular expressions for validation
var (
	// alphanumeric with underscore and hyphen
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// Bech32 address format (paw1...)
	bech32Regex = regexp.MustCompile(`^[a-z]{3,10}1[a-z0-9]{38,100}$`)

	// Hex string (0x prefix optional)
	hexRegex = regexp.MustCompile(`^(0x)?[0-9a-fA-F]+$`)

	// Numeric string (positive decimal)
	numericRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

	// Order ID format
	orderIDRegex = regexp.MustCompile(`^ORD-[0-9a-f]{16}$`)

	// Trade ID format
	tradeIDRegex = regexp.MustCompile(`^TRD-[0-9a-f]{16}$`)

	// Swap ID format
	swapIDRegex = regexp.MustCompile(`^SWAP-[0-9a-f]{16}$`)

	// Pool ID format
	poolIDRegex = regexp.MustCompile(`^pool-[0-9]+$`)
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors holds multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (v *ValidationErrors) Add(field, message string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

func (v *ValidationErrors) Error() string {
	if !v.HasErrors() {
		return ""
	}
	var sb strings.Builder
	for i, err := range v.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return sb.String()
}

// ===================  Input Sanitization ====================

// SanitizeString removes potentially dangerous characters and HTML
func SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	// Escape HTML entities
	input = html.EscapeString(input)
	// Trim whitespace
	input = strings.TrimSpace(input)
	return input
}

// SanitizeURL validates and sanitizes URL input
func SanitizeURL(input string) (string, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("invalid URL format")
	}

	// Only allow http and https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("only http and https schemes are allowed")
	}

	return parsed.String(), nil
}

// SanitizeJSON escapes JSON strings to prevent injection
func SanitizeJSON(input string) string {
	// Remove control characters
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, input)
	return input
}

// =================== Username Validation ===================

// ValidateUsername validates username format and length
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)

	if len(username) < MinUsernameLength {
		return fmt.Errorf("username must be at least %d characters", MinUsernameLength)
	}

	if len(username) > MaxUsernameLength {
		return fmt.Errorf("username must not exceed %d characters", MaxUsernameLength)
	}

	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}

	// Check for reserved usernames
	reserved := []string{"admin", "root", "system", "api", "paw", "test"}
	lowerUsername := strings.ToLower(username)
	for _, r := range reserved {
		if lowerUsername == r {
			return fmt.Errorf("username is reserved")
		}
	}

	return nil
}

// =================== Password Validation ===================

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password must not exceed %d characters", MaxPasswordLength)
	}

	// Check for at least one uppercase, one lowercase, one digit
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}

// =================== Address Validation ===================

// ValidateAddress validates blockchain address format
func ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address is required")
	}

	address = strings.TrimSpace(address)

	if len(address) > MaxAddressLength {
		return fmt.Errorf("address too long")
	}

	// Try to parse as Cosmos SDK address
	_, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		// Check if it matches bech32 format at least
		if !bech32Regex.MatchString(address) {
			return fmt.Errorf("invalid address format")
		}
	}

	return nil
}

// =================== Amount Validation ===================

// ValidateAmount validates amount strings
func ValidateAmount(amount string) error {
	if amount == "" {
		return fmt.Errorf("amount is required")
	}

	amount = strings.TrimSpace(amount)

	if len(amount) > MaxAmountLength {
		return fmt.Errorf("amount too long")
	}

	if !numericRegex.MatchString(amount) {
		return fmt.Errorf("amount must be a positive number")
	}

	// Try to parse as SDK Dec (using LegacyNewDecFromStr for SDK v0.50+)
	_, err := math.LegacyNewDecFromStr(amount)
	if err != nil {
		return fmt.Errorf("invalid amount format: %w", err)
	}

	return nil
}

// ValidatePriceAmount validates trading price/amount
func ValidatePriceAmount(value float64) error {
	if value <= 0 {
		return fmt.Errorf("value must be positive")
	}

	if value > 1e15 {
		return fmt.Errorf("value too large")
	}

	return nil
}

// =================== Token/Denom Validation ===================

// ValidateDenom validates token denomination
func ValidateDenom(denom string) error {
	if denom == "" {
		return fmt.Errorf("denom is required")
	}

	denom = strings.TrimSpace(denom)

	if len(denom) < 3 || len(denom) > 128 {
		return fmt.Errorf("denom must be between 3 and 128 characters")
	}

	// Check if it's alphanumeric with some special chars
	validDenom := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9/-]{2,127}$`)
	if !validDenom.MatchString(denom) {
		return fmt.Errorf("invalid denom format")
	}

	return nil
}

// =================== ID Validation ===================

// ValidateOrderID validates order ID format
func ValidateOrderID(orderID string) error {
	if orderID == "" {
		return fmt.Errorf("order ID is required")
	}

	if len(orderID) > MaxOrderIDLength {
		return fmt.Errorf("order ID too long")
	}

	if !orderIDRegex.MatchString(orderID) {
		return fmt.Errorf("invalid order ID format")
	}

	return nil
}

// ValidateSwapID validates swap ID format
func ValidateSwapID(swapID string) error {
	if swapID == "" {
		return fmt.Errorf("swap ID is required")
	}

	if len(swapID) > MaxSwapIDLength {
		return fmt.Errorf("swap ID too long")
	}

	if !swapIDRegex.MatchString(swapID) {
		return fmt.Errorf("invalid swap ID format")
	}

	return nil
}

// ValidatePoolID validates pool ID format
func ValidatePoolID(poolID string) error {
	if poolID == "" {
		return fmt.Errorf("pool ID is required")
	}

	if len(poolID) > MaxPoolIDLength {
		return fmt.Errorf("pool ID too long")
	}

	if !poolIDRegex.MatchString(poolID) {
		return fmt.Errorf("invalid pool ID format")
	}

	return nil
}

// =================== Hash Validation ===================

// ValidateHash validates hex hash strings
func ValidateHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("hash is required")
	}

	hash = strings.TrimSpace(hash)

	// Remove 0x prefix if present
	hash = strings.TrimPrefix(hash, "0x")

	if len(hash) > MaxHashLength {
		return fmt.Errorf("hash too long")
	}

	if !hexRegex.MatchString(hash) {
		return fmt.Errorf("invalid hash format")
	}

	return nil
}

// ValidateSecret validates secret format for atomic swaps
func ValidateSecret(secret string) error {
	if secret == "" {
		return fmt.Errorf("secret is required")
	}

	secret = strings.TrimSpace(secret)

	if len(secret) > MaxSecretLength {
		return fmt.Errorf("secret too long")
	}

	if !hexRegex.MatchString(secret) {
		return fmt.Errorf("invalid secret format")
	}

	return nil
}

// =================== Memo Validation ===================

// ValidateMemo validates transaction memo
func ValidateMemo(memo string) error {
	if len(memo) > MaxMemoLength {
		return fmt.Errorf("memo must not exceed %d characters", MaxMemoLength)
	}

	// Check for null bytes and control characters
	for _, r := range memo {
		if r == 0 || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			return fmt.Errorf("memo contains invalid characters")
		}
	}

	return nil
}

// =================== Pagination Validation ===================

// ValidatePagination validates and sanitizes pagination parameters
func ValidatePagination(params *PaginationParams) error {
	if params.Page < 1 {
		params.Page = 1
	}

	if params.PageSize < 1 {
		params.PageSize = 20
	}

	if params.PageSize > 100 {
		params.PageSize = 100
	}

	params.Offset = (params.Page - 1) * params.PageSize

	return nil
}

// =================== Query Parameter Validation ===================

// ValidateLimit validates limit query parameter
func ValidateLimit(limitStr string, defaultLimit, maxLimit int) int {
	limit := defaultLimit
	if limitStr != "" {
		if regexp.MustCompile(`^[0-9]+$`).MatchString(limitStr) {
			fmt.Sscanf(limitStr, "%d", &limit)
		}
	}

	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return limit
}

// ValidateHeight validates block height
func ValidateHeight(heightStr string) (int64, error) {
	if heightStr == "" {
		return 0, fmt.Errorf("height is required")
	}

	if !regexp.MustCompile(`^[0-9]+$`).MatchString(heightStr) {
		return 0, fmt.Errorf("height must be a positive integer")
	}

	var height int64
	_, err := fmt.Sscanf(heightStr, "%d", &height)
	if err != nil {
		return 0, fmt.Errorf("invalid height format")
	}

	if height < 0 {
		return 0, fmt.Errorf("height must be non-negative")
	}

	if height > 1e12 { // Sanity check
		return 0, fmt.Errorf("height too large")
	}

	return height, nil
}

// =================== Request Validation ===================

// ValidateRegisterRequest validates registration request
func ValidateRegisterRequest(req *RegisterRequest) error {
	errors := &ValidationErrors{}

	if err := ValidateUsername(req.Username); err != nil {
		errors.Add("username", err.Error())
	}

	if err := ValidatePassword(req.Password); err != nil {
		errors.Add("password", err.Error())
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.Username = SanitizeString(req.Username)

	return nil
}

// ValidateLoginRequest validates login request
func ValidateLoginRequest(req *LoginRequest) error {
	errors := &ValidationErrors{}

	if req.Username == "" {
		errors.Add("username", "username is required")
	} else if len(req.Username) > MaxUsernameLength {
		errors.Add("username", "username too long")
	}

	if req.Password == "" {
		errors.Add("password", "password is required")
	} else if len(req.Password) > MaxPasswordLength {
		errors.Add("password", "password too long")
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.Username = SanitizeString(req.Username)

	return nil
}

// ValidateCreateOrderRequest validates order creation request
func ValidateCreateOrderRequest(req *CreateOrderRequest) error {
	errors := &ValidationErrors{}

	if req.OrderType != "buy" && req.OrderType != "sell" {
		errors.Add("order_type", "order_type must be 'buy' or 'sell'")
	}

	if err := ValidatePriceAmount(req.Price); err != nil {
		errors.Add("price", err.Error())
	}

	if err := ValidatePriceAmount(req.Amount); err != nil {
		errors.Add("amount", err.Error())
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ValidateSendTokensRequest validates token send request
func ValidateSendTokensRequest(req *SendTokensRequest) error {
	errors := &ValidationErrors{}

	if err := ValidateAddress(req.ToAddress); err != nil {
		errors.Add("to_address", err.Error())
	}

	if err := ValidateAmount(req.Amount); err != nil {
		errors.Add("amount", err.Error())
	}

	if err := ValidateDenom(req.Denom); err != nil {
		errors.Add("denom", err.Error())
	}

	if req.Memo != "" {
		if err := ValidateMemo(req.Memo); err != nil {
			errors.Add("memo", err.Error())
		}
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.ToAddress = SanitizeString(req.ToAddress)
	req.Memo = SanitizeString(req.Memo)

	return nil
}

// ValidatePrepareSwapRequest validates swap preparation request
func ValidatePrepareSwapRequest(req *PrepareSwapRequest) error {
	errors := &ValidationErrors{}

	if err := ValidateAddress(req.CounterpartyAddress); err != nil {
		errors.Add("counterparty_address", err.Error())
	}

	if err := ValidateAmount(req.SendAmount); err != nil {
		errors.Add("send_amount", err.Error())
	}

	if err := ValidateDenom(req.SendDenom); err != nil {
		errors.Add("send_denom", err.Error())
	}

	if err := ValidateAmount(req.ReceiveAmount); err != nil {
		errors.Add("receive_amount", err.Error())
	}

	if err := ValidateDenom(req.ReceiveDenom); err != nil {
		errors.Add("receive_denom", err.Error())
	}

	if req.HashLock != "" {
		if err := ValidateHash(req.HashLock); err != nil {
			errors.Add("hash_lock", err.Error())
		}
	}

	if req.TimeLockDuration < 0 {
		errors.Add("timelock_duration", "timelock_duration must be non-negative")
	}

	if req.TimeLockDuration > 86400*30 { // Max 30 days
		errors.Add("timelock_duration", "timelock_duration too long (max 30 days)")
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.CounterpartyAddress = SanitizeString(req.CounterpartyAddress)

	return nil
}

// ValidateCommitSwapRequest validates swap commit request
func ValidateCommitSwapRequest(req *CommitSwapRequest) error {
	errors := &ValidationErrors{}

	if err := ValidateSwapID(req.SwapID); err != nil {
		errors.Add("swap_id", err.Error())
	}

	if req.Secret != "" {
		if err := ValidateSecret(req.Secret); err != nil {
			errors.Add("secret", err.Error())
		}
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.SwapID = SanitizeString(req.SwapID)
	req.Secret = SanitizeString(req.Secret)

	return nil
}

// ValidateAddLiquidityRequest validates add liquidity request
func ValidateAddLiquidityRequest(req *AddLiquidityRequest) error {
	errors := &ValidationErrors{}

	if err := ValidatePoolID(req.PoolID); err != nil {
		errors.Add("pool_id", err.Error())
	}

	if err := ValidateAmount(req.AmountA); err != nil {
		errors.Add("amount_a", err.Error())
	}

	if err := ValidateAmount(req.AmountB); err != nil {
		errors.Add("amount_b", err.Error())
	}

	if req.Slippage < 0 || req.Slippage > 1 {
		errors.Add("slippage", "slippage must be between 0 and 1")
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.PoolID = SanitizeString(req.PoolID)

	return nil
}

// ValidateRemoveLiquidityRequest validates remove liquidity request
func ValidateRemoveLiquidityRequest(req *RemoveLiquidityRequest) error {
	errors := &ValidationErrors{}

	if err := ValidatePoolID(req.PoolID); err != nil {
		errors.Add("pool_id", err.Error())
	}

	if err := ValidateAmount(req.Shares); err != nil {
		errors.Add("shares", err.Error())
	}

	if req.MinAmountA != "" {
		if err := ValidateAmount(req.MinAmountA); err != nil {
			errors.Add("min_amount_a", err.Error())
		}
	}

	if req.MinAmountB != "" {
		if err := ValidateAmount(req.MinAmountB); err != nil {
			errors.Add("min_amount_b", err.Error())
		}
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.PoolID = SanitizeString(req.PoolID)

	return nil
}

// ValidateVerifyProofRequest validates proof verification request
func ValidateVerifyProofRequest(req *VerifyProofRequest) error {
	errors := &ValidationErrors{}

	if err := ValidateHash(req.TxHash); err != nil {
		errors.Add("tx_hash", err.Error())
	}

	if req.Height <= 0 {
		errors.Add("height", "height must be positive")
	}

	if len(req.Proof) == 0 {
		errors.Add("proof", "proof is required")
	}

	for i, p := range req.Proof {
		if err := ValidateHash(p); err != nil {
			errors.Add(fmt.Sprintf("proof[%d]", i), err.Error())
		}
	}

	if err := ValidateHash(req.BlockHash); err != nil {
		errors.Add("block_hash", err.Error())
	}

	if errors.HasErrors() {
		return errors
	}

	// Sanitize
	req.TxHash = SanitizeString(req.TxHash)
	req.BlockHash = SanitizeString(req.BlockHash)
	for i := range req.Proof {
		req.Proof[i] = SanitizeString(req.Proof[i])
	}

	return nil
}

// =================== Helper Function for Gin Context ===================

// ValidateAndBindJSON validates and binds JSON with size limit
func ValidateAndBindJSON(c *gin.Context, obj interface{}) error {
	// Check content length
	if c.Request.ContentLength > MaxRequestSize {
		return fmt.Errorf("request body too large (max %d bytes)", MaxRequestSize)
	}

	// Bind JSON
	if err := c.ShouldBindJSON(obj); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

// GetUserFromContext safely retrieves user info from context
func GetUserFromContext(c *gin.Context) (userID, username, address string, err error) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return "", "", "", fmt.Errorf("user not authenticated")
	}

	usernameVal, _ := c.Get("username")
	addressVal, _ := c.Get("address")

	userID, _ = userIDVal.(string)
	username, _ = usernameVal.(string)
	address, _ = addressVal.(string)

	if userID == "" {
		return "", "", "", fmt.Errorf("invalid user context")
	}

	return userID, username, address, nil
}
