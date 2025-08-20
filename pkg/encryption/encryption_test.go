package encryption

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool // whether encryptor should be created successfully
	}{
		{
			name:     "valid password",
			password: "test-password-123",
			want:     true,
		},
		{
			name:     "empty password",
			password: "",
			want:     true, // Empty password is allowed, though not recommended
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 256),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor := NewEncryptor(tt.password)
			if tt.want && encryptor == nil {
				t.Error("NewEncryptor() returned nil, expected valid encryptor")
			}
			if !tt.want && encryptor != nil {
				t.Error("NewEncryptor() returned encryptor, expected nil")
			}
		})
	}
}

func TestNewEncryptorFromEnv(t *testing.T) {
	encryptor, err := NewEncryptorFromEnv()
	if err != nil {
		t.Fatalf("NewEncryptorFromEnv() error = %v", err)
	}
	if encryptor == nil {
		t.Error("NewEncryptorFromEnv() returned nil encryptor")
	}
}

func TestEncryptor_Encrypt(t *testing.T) {
	encryptor := NewEncryptor("test-password")
	
	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "valid plaintext",
			plaintext: []byte("Hello, World!"),
			wantErr:   false,
		},
		{
			name:      "empty plaintext",
			plaintext: []byte(""),
			wantErr:   true, // Should fail for empty plaintext
		},
		{
			name:      "long plaintext",
			plaintext: []byte(strings.Repeat("Hello World! ", 1000)),
			wantErr:   false,
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptor.Encrypt(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if encrypted == nil {
					t.Error("Encrypt() returned nil encrypted data")
					return
				}
				
				// Verify encrypted data structure
				if len(encrypted.Salt) != SaltSize {
					t.Errorf("Encrypt() salt size = %v, want %v", len(encrypted.Salt), SaltSize)
				}
				
				if len(encrypted.Nonce) != NonceSize {
					t.Errorf("Encrypt() nonce size = %v, want %v", len(encrypted.Nonce), NonceSize)
				}
				
				if len(encrypted.Ciphertext) == 0 {
					t.Error("Encrypt() ciphertext is empty")
				}
				
				if encrypted.Version != 1 {
					t.Errorf("Encrypt() version = %v, want 1", encrypted.Version)
				}
			}
		})
	}
}

func TestEncryptor_Decrypt(t *testing.T) {
	encryptor := NewEncryptor("test-password")
	
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "small text",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "json data",
			plaintext: []byte(`{"key": "value", "number": 42}`),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "unicode text",
			plaintext: []byte("Hello 世界 🌍"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt the data
			encrypted, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			
			// Decrypt the data
			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			
			// Verify the decrypted data matches original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptor_Decrypt_InvalidData(t *testing.T) {
	encryptor := NewEncryptor("test-password")
	
	tests := []struct {
		name string
		data *EncryptedData
	}{
		{
			name: "nil data",
			data: nil,
		},
		{
			name: "invalid version",
			data: &EncryptedData{
				Version:    99,
				Salt:       make([]byte, SaltSize),
				Nonce:      make([]byte, NonceSize),
				Ciphertext: []byte("invalid"),
			},
		},
		{
			name: "invalid salt size",
			data: &EncryptedData{
				Version:    1,
				Salt:       make([]byte, 8), // Wrong size
				Nonce:      make([]byte, NonceSize),
				Ciphertext: []byte("invalid"),
			},
		},
		{
			name: "invalid nonce size",
			data: &EncryptedData{
				Version:    1,
				Salt:       make([]byte, SaltSize),
				Nonce:      make([]byte, 8), // Wrong size
				Ciphertext: []byte("invalid"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.data)
			if err == nil {
				t.Error("Decrypt() expected error for invalid data, got nil")
			}
		})
	}
}

func TestEncryptor_EncryptString(t *testing.T) {
	encryptor := NewEncryptor("test-password")
	
	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "valid string",
			plaintext: "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "empty string",
			plaintext: "",
			wantErr:   true,
		},
		{
			name:      "unicode string",
			plaintext: "Hello 世界 🌍",
			wantErr:   false,
		},
		{
			name:      "long string",
			plaintext: strings.Repeat("Hello World! ", 1000),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptor.EncryptString(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if encrypted == "" {
					t.Error("EncryptString() returned empty string")
					return
				}
				
				// Should be base64 encoded
				if len(encrypted) < 20 { // Minimum reasonable length
					t.Error("EncryptString() returned suspiciously short encrypted string")
				}
				
				// Test round trip
				decrypted, err := encryptor.DecryptString(encrypted)
				if err != nil {
					t.Fatalf("DecryptString() error = %v", err)
				}
				
				if decrypted != tt.plaintext {
					t.Errorf("DecryptString() = %v, want %v", decrypted, tt.plaintext)
				}
			}
		})
	}
}

func TestEncryptor_DecryptString_InvalidData(t *testing.T) {
	encryptor := NewEncryptor("test-password")
	
	tests := []struct {
		name      string
		encrypted string
	}{
		{
			name:      "invalid base64",
			encrypted: "not-valid-base64!@#",
		},
		{
			name:      "empty string",
			encrypted: "",
		},
		{
			name:      "invalid format",
			encrypted: "dGVzdA==", // Valid base64 but wrong format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.DecryptString(tt.encrypted)
			if err == nil {
				t.Error("DecryptString() expected error for invalid data, got nil")
			}
		})
	}
}

func TestEncryptor_DifferentPasswords(t *testing.T) {
	encryptor1 := NewEncryptor("password1")
	encryptor2 := NewEncryptor("password2")
	
	plaintext := []byte("secret data")
	
	// Encrypt with first encryptor
	encrypted, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Try to decrypt with second encryptor (different password)
	_, err = encryptor2.Decrypt(encrypted)
	if err == nil {
		t.Error("Decrypt() with wrong password should fail, but succeeded")
	}
	
	// Decrypt with correct encryptor should work
	decrypted, err := encryptor1.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() with correct password error = %v", err)
	}
	
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid strong password",
			password: "MySecure123!",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Abc1!",
			wantErr:  true,
		},
		{
			name:     "no uppercase",
			password: "mysecure123!",
			wantErr:  true,
		},
		{
			name:     "no lowercase",
			password: "MYSECURE123!",
			wantErr:  true,
		},
		{
			name:     "no digit",
			password: "MySecurePass!",
			wantErr:  true,
		},
		{
			name:     "no special character",
			password: "MySecurePass123",
			wantErr:  true,
		},
		{
			name:     "valid long password",
			password: "MyVeryLongAndSecurePassword123!@#",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "minimum length",
			length: 12,
		},
		{
			name:   "short length gets upgraded",
			length: 8, // Should be upgraded to 12
		},
		{
			name:   "medium length",
			length: 20,
		},
		{
			name:   "long length",
			length: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateSecurePassword(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecurePassword() error = %v", err)
			}
			
			expectedLength := tt.length
			if expectedLength < 12 {
				expectedLength = 12
			}
			
			if len(password) != expectedLength {
				t.Errorf("GenerateSecurePassword() length = %v, want %v", len(password), expectedLength)
			}
			
			// Generated password should pass validation
			if err := ValidatePassword(password); err != nil {
				t.Errorf("Generated password failed validation: %v, password: %s", err, password)
			}
			
			// Generate another password and ensure they're different
			password2, err := GenerateSecurePassword(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecurePassword() second call error = %v", err)
			}
			
			if password == password2 {
				t.Error("GenerateSecurePassword() returned identical passwords - not cryptographically secure")
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify cryptographic constants are reasonable
	if KeySize != 32 {
		t.Errorf("KeySize = %v, want 32 (AES-256)", KeySize)
	}
	
	if SaltSize != 16 {
		t.Errorf("SaltSize = %v, want 16", SaltSize)
	}
	
	if NonceSize != 12 {
		t.Errorf("NonceSize = %v, want 12 (GCM standard)", NonceSize)
	}
	
	if PBKDF2Iterations < 100000 {
		t.Errorf("PBKDF2Iterations = %v, want at least 100000", PBKDF2Iterations)
	}
}

// Benchmark tests
func BenchmarkEncrypt(b *testing.B) {
	encryptor := NewEncryptor("test-password")
	data := []byte(strings.Repeat("Hello, World! ", 100))
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptor.Encrypt(data)
		if err != nil {
			b.Fatalf("Encrypt() error = %v", err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	encryptor := NewEncryptor("test-password")
	data := []byte(strings.Repeat("Hello, World! ", 100))
	
	// Encrypt once
	encrypted, err := encryptor.Encrypt(data)
	if err != nil {
		b.Fatalf("Encrypt() error = %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptor.Decrypt(encrypted)
		if err != nil {
			b.Fatalf("Decrypt() error = %v", err)
		}
	}
}

func BenchmarkEncryptString(b *testing.B) {
	encryptor := NewEncryptor("test-password")
	data := strings.Repeat("Hello, World! ", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptor.EncryptString(data)
		if err != nil {
			b.Fatalf("EncryptString() error = %v", err)
		}
	}
}

func BenchmarkDecryptString(b *testing.B) {
	encryptor := NewEncryptor("test-password")
	data := strings.Repeat("Hello, World! ", 100)
	
	// Encrypt once
	encrypted, err := encryptor.EncryptString(data)
	if err != nil {
		b.Fatalf("EncryptString() error = %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptor.DecryptString(encrypted)
		if err != nil {
			b.Fatalf("DecryptString() error = %v", err)
		}
	}
}