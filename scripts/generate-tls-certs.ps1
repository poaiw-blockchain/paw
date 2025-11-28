# PAW Blockchain TLS Certificate Generation Script (PowerShell)
# This script generates TLS certificates for the API server on Windows
#
# Usage:
#   .\scripts\generate-tls-certs.ps1 [-Environment dev]
#
# Parameters:
#   -Environment: "dev" (default) or "staging"
#
# For PRODUCTION, use certificates from a trusted CA like Let's Encrypt
# See PRODUCTION_CERTIFICATES.md for instructions

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("dev", "staging")]
    [string]$Environment = "dev"
)

$ErrorActionPreference = "Stop"

# Script directories
$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path $ScriptDir
$CertsDir = Join-Path $ProjectRoot "certs"
$EnvCertsDir = Join-Path $CertsDir $Environment

# Certificate parameters
$DaysValid = 365
$KeySize = 2048
$Country = "US"
$State = "California"
$Locality = "San Francisco"
$Organization = "PAW Blockchain"
$OrganizationalUnit = "Engineering"
$CommonName = "localhost"

# Color functions
function Write-Header {
    param([string]$Message)
    Write-Host "========================================" -ForegroundColor Green
    Write-Host $Message -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
}

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

# Check if OpenSSL is available
function Test-Dependencies {
    Write-Info "Checking for OpenSSL..."

    # Try to find OpenSSL in common locations
    $opensslPaths = @(
        "C:\Program Files\OpenSSL-Win64\bin\openssl.exe",
        "C:\Program Files (x86)\OpenSSL-Win32\bin\openssl.exe",
        "C:\OpenSSL-Win64\bin\openssl.exe",
        "openssl.exe"  # In PATH
    )

    foreach ($path in $opensslPaths) {
        if (Get-Command $path -ErrorAction SilentlyContinue) {
            $script:OpenSSL = $path
            $version = & $script:OpenSSL version
            Write-Info "Found OpenSSL: $version"
            return $true
        }
    }

    # Check if running in  Bash environment
    if (Test-Path "C:\Program Files\\usr\bin\openssl.exe") {
        $script:OpenSSL = "C:\Program Files\\usr\bin\openssl.exe"
        $version = & $script:OpenSSL version
        Write-Info "Found OpenSSL (): $version"
        return $true
    }

    Write-Error "OpenSSL is not installed or not found in PATH"
    Write-Host ""
    Write-Host "Please install OpenSSL for Windows:"
    Write-Host "  1. Download from: https://slproweb.com/products/Win32OpenSSL.html"
    Write-Host "  2. Install 'Win64 OpenSSL v3.x.x' (or latest version)"
    Write-Host "  3. Add OpenSSL bin directory to PATH"
    Write-Host ""
    Write-Host "Alternatively, use  Bash which includes OpenSSL:"
    Write-Host "  Run: ./scripts/generate-tls-certs.sh from  Bash"
    exit 1
}

# Create directories
function Initialize-Directories {
    if (-not (Test-Path $CertsDir)) {
        New-Item -ItemType Directory -Path $CertsDir | Out-Null
        Write-Info "Created directory: $CertsDir"
    }

    if (-not (Test-Path $EnvCertsDir)) {
        New-Item -ItemType Directory -Path $EnvCertsDir | Out-Null
        Write-Info "Created directory: $EnvCertsDir"
    }
}

# Generate OpenSSL configuration
function New-OpenSSLConfig {
    $configFile = Join-Path $EnvCertsDir "openssl.cnf"

    $config = @"
[req]
default_bits = $KeySize
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C = $Country
ST = $State
L = $Locality
O = $Organization
OU = $OrganizationalUnit
CN = $CommonName
emailAddress = admin@pawchain.io

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[v3_ca]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints = critical, CA:true
keyUsage = critical, digitalSignature, cRLSign, keyCertSign

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
DNS.3 = 127.0.0.1
DNS.4 = pawchain.local
DNS.5 = *.pawchain.local
IP.1 = 127.0.0.1
IP.2 = ::1
"@

    if ($Environment -eq "staging") {
        $config += @"

DNS.6 = staging.pawchain.io
DNS.7 = *.staging.pawchain.io
"@
    }

    $config | Out-File -FilePath $configFile -Encoding ASCII
    Write-Info "Generated OpenSSL configuration: $configFile"

    return $configFile
}

# Generate Certificate Authority
function New-CertificateAuthority {
    param([string]$ConfigFile)

    Write-Header "Generating Certificate Authority (CA)"

    $caKey = Join-Path $EnvCertsDir "ca-key.pem"
    $caCert = Join-Path $EnvCertsDir "ca-cert.pem"

    # Generate CA private key
    & $script:OpenSSL genrsa -out $caKey 4096 2>&1 | Out-Null
    Write-Info "Generated CA private key: $caKey"

    # Generate CA certificate
    $caSubject = "/C=$Country/ST=$State/L=$Locality/O=$Organization/OU=$OrganizationalUnit CA/CN=PAW Blockchain CA"
    & $script:OpenSSL req -new -x509 -sha256 `
        -key $caKey `
        -out $caCert `
        -days ($DaysValid * 2) `
        -config $ConfigFile `
        -extensions v3_ca `
        -subj $caSubject 2>&1 | Out-Null

    Write-Info "Generated CA certificate: $caCert"
}

# Generate Server Certificate
function New-ServerCertificate {
    param([string]$ConfigFile)

    Write-Header "Generating Server Certificate"

    $serverKey = Join-Path $EnvCertsDir "server-key.pem"
    $serverCsr = Join-Path $EnvCertsDir "server-csr.pem"
    $serverCert = Join-Path $EnvCertsDir "server-cert.pem"
    $caKey = Join-Path $EnvCertsDir "ca-key.pem"
    $caCert = Join-Path $EnvCertsDir "ca-cert.pem"

    # Generate server private key
    & $script:OpenSSL genrsa -out $serverKey $KeySize 2>&1 | Out-Null
    Write-Info "Generated server private key: $serverKey"

    # Generate CSR
    & $script:OpenSSL req -new -sha256 `
        -key $serverKey `
        -out $serverCsr `
        -config $ConfigFile 2>&1 | Out-Null

    Write-Info "Generated certificate signing request: $serverCsr"

    # Sign CSR with CA
    & $script:OpenSSL x509 -req -sha256 `
        -in $serverCsr `
        -CA $caCert `
        -CAkey $caKey `
        -CAcreateserial `
        -out $serverCert `
        -days $DaysValid `
        -extensions v3_req `
        -extfile $ConfigFile 2>&1 | Out-Null

    Write-Info "Generated server certificate: $serverCert"

    # Clean up CSR
    Remove-Item $serverCsr -ErrorAction SilentlyContinue
}

# Verify certificates
function Test-Certificates {
    Write-Header "Verifying Certificates"

    $serverCert = Join-Path $EnvCertsDir "server-cert.pem"
    $caCert = Join-Path $EnvCertsDir "ca-cert.pem"

    # Verify server certificate
    $verifyResult = & $script:OpenSSL verify -CAfile $caCert $serverCert 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Info "✓ Server certificate verification: PASSED"
    } else {
        Write-Error "✗ Server certificate verification: FAILED"
        Write-Host $verifyResult
        exit 1
    }

    # Display certificate details
    Write-Host ""
    Write-Info "Server Certificate Details:"
    & $script:OpenSSL x509 -in $serverCert -noout -subject -issuer -dates

    Write-Host ""
    Write-Info "Subject Alternative Names (SANs):"
    & $script:OpenSSL x509 -in $serverCert -noout -text | Select-String -Pattern "DNS:" -Context 0,3
}

# Create configuration example
function New-ConfigExample {
    Write-Header "Creating Configuration Example"

    $configExample = Join-Path $EnvCertsDir "server-config-example.yaml"

    $exampleConfig = @"
# PAW API Server TLS Configuration Example
#
# Add these settings to your config.yaml or use environment variables

api:
  # Enable TLS
  tls_enabled: true

  # Path to server certificate
  tls_cert_file: "$EnvCertsDir\server-cert.pem"

  # Path to server private key
  tls_key_file: "$EnvCertsDir\server-key.pem"

  # Server address
  address: "0.0.0.0:8443"

# Environment Variables (alternative to YAML config):
#   `$env:PAW_API_TLS_ENABLED = "true"
#   `$env:PAW_API_TLS_CERT_FILE = "$EnvCertsDir\server-cert.pem"
#   `$env:PAW_API_TLS_KEY_FILE = "$EnvCertsDir\server-key.pem"
#   `$env:PAW_API_ADDRESS = "0.0.0.0:8443"
"@

    $exampleConfig | Out-File -FilePath $configExample -Encoding UTF8
    Write-Info "Created configuration example: $configExample"
}

# Display usage instructions
function Show-Instructions {
    Write-Header "TLS Certificates Generated Successfully!"

    Write-Host ""
    Write-Host "Certificate Files:" -ForegroundColor Green
    Write-Host "  CA Certificate:     $EnvCertsDir\ca-cert.pem"
    Write-Host "  Server Certificate: $EnvCertsDir\server-cert.pem"
    Write-Host "  Server Private Key: $EnvCertsDir\server-key.pem"
    Write-Host ""

    Write-Host "Next Steps:" -ForegroundColor Green
    Write-Host ""
    Write-Host "1. Configure the API server to use these certificates:"
    Write-Host "   See: $EnvCertsDir\server-config-example.yaml"
    Write-Host ""
    Write-Host "2. For development/testing, trust the CA certificate:"
    Write-Host "   a. Open Certificate Manager:"
    Write-Host "      - Press Win+R, type 'certmgr.msc', press Enter"
    Write-Host "   b. Navigate to: Trusted Root Certification Authorities > Certificates"
    Write-Host "   c. Right-click > All Tasks > Import"
    Write-Host "   d. Import: $EnvCertsDir\ca-cert.pem"
    Write-Host ""
    Write-Host "   OR use PowerShell (requires Admin):"
    Write-Host "   Import-Certificate -FilePath '$EnvCertsDir\ca-cert.pem' -CertStoreLocation Cert:\LocalMachine\Root"
    Write-Host ""
    Write-Host "3. Test the API server with curl:"
    Write-Host "   curl --cacert '$EnvCertsDir\ca-cert.pem' https://localhost:8443/api/v1/health"
    Write-Host ""

    Write-Warning "IMPORTANT: These are self-signed certificates for $Environment use only!"
    Write-Warning "For PRODUCTION, use certificates from a trusted CA (e.g., Let's Encrypt)"
    Write-Host ""
    Write-Host "See PRODUCTION_CERTIFICATES.md for production certificate instructions."
}

# Create ignore
function New-GitIgnore {
    $gitignore = Join-Path $CertsDir "ignore"

    if (-not (Test-Path $gitignore)) {
        $ignoreContent = @"
# Ignore all certificate files for security
*.pem
*.key
*.crt
*.csr
*.srl

# Keep only documentation and scripts
!ignore
!README.md
"@
        $ignoreContent | Out-File -FilePath $gitignore -Encoding UTF8
        Write-Info "Created ignore: $gitignore"
    }
}

# Main execution
function Main {
    Write-Header "PAW Blockchain TLS Certificate Generator (PowerShell)"
    Write-Host ""
    Write-Info "Environment: $Environment"
    Write-Info "Certificate validity: $DaysValid days"
    Write-Host ""

    Test-Dependencies
    Initialize-Directories
    New-GitIgnore

    $configFile = New-OpenSSLConfig
    New-CertificateAuthority -ConfigFile $configFile
    New-ServerCertificate -ConfigFile $configFile
    Test-Certificates
    New-ConfigExample
    Show-Instructions
}

# Run main function
try {
    Main
} catch {
    Write-Error "An error occurred: $_"
    exit 1
}
