package form_validator

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Error("Expected non-nil Validator")
	}
	if len(v.Errors) != 0 {
		t.Error("Expected empty errors map")
	}
}

func TestValidator_SetValue(t *testing.T) {
	v := New()
	v.SetValue("name", "John")

	if got := v.GetValue("name"); got != "John" {
		t.Errorf("GetValue() = %v, want %v", got, "John")
	}
}

func TestValidator_String(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       string
		validations []ValidationFunc
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "valid required string",
			field:       "name",
			value:       "John",
			validations: []ValidationFunc{Required},
			wantErr:     false,
		},
		{
			name:        "invalid required string",
			field:       "name",
			value:       "",
			validations: []ValidationFunc{Required},
			wantErr:     true,
			errMessage:  "This field is required",
		},
		{
			name:        "valid email",
			field:       "email",
			value:       "test@example.com",
			validations: []ValidationFunc{Email},
			wantErr:     false,
		},
		{
			name:        "invalid email",
			field:       "email",
			value:       "invalid-email",
			validations: []ValidationFunc{Email},
			wantErr:     true,
			errMessage:  "Please enter a valid email address",
		},
		{
			name:        "valid string length",
			field:       "username",
			value:       "johndoe",
			validations: []ValidationFunc{MinLength(3), MaxLength(10)},
			wantErr:     false,
		},
		{
			name:        "string too short",
			field:       "username",
			value:       "jo",
			validations: []ValidationFunc{MinLength(3)},
			wantErr:     true,
			errMessage:  "This field must be at least 3 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.SetValue(tt.field, tt.value)
			v.String(tt.field, tt.validations...)

			if tt.wantErr {
				if err, ok := v.Errors[tt.field]; !ok {
					t.Errorf("Expected error for field %s", tt.field)
				} else if err != tt.errMessage {
					t.Errorf("Expected error message %q, got %q", tt.errMessage, err)
				}
			} else {
				if err, ok := v.Errors[tt.field]; ok {
					t.Errorf("Unexpected error for field %s: %s", tt.field, err)
				}
			}
		})
	}
}

func TestValidator_Int(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       string
		validations []ValidationFunc
		want        int64
		wantErr     bool
	}{
		{
			name:    "valid integer",
			field:   "age",
			value:   "25",
			want:    25,
			wantErr: false,
		},
		{
			name:    "invalid integer",
			field:   "age",
			value:   "not-a-number",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.SetValue(tt.field, tt.value)
			got := v.Int(tt.field, tt.validations...)

			if tt.wantErr {
				if _, ok := v.Errors[tt.field]; !ok {
					t.Errorf("Expected error for field %s", tt.field)
				}
			} else {
				if got != tt.want {
					t.Errorf("Int() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestValidator_Image(t *testing.T) {
	// Create a minimal valid JPEG file content.
	jpegContent := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0xFF, 0xD9,
	}

	tests := []struct {
		name     string
		field    string
		filename string
		content  []byte
		config   FileValidationConfig
		wantErr  bool
	}{
		{
			name:     "valid image",
			field:    "avatar",
			filename: "test.jpg",
			content:  jpegContent,
			config:   ImageConfig(1 * MB),
			wantErr:  false,
		},
		{
			name:     "invalid extension",
			field:    "avatar",
			filename: "test.txt",
			content:  []byte("text content"),
			config:   ImageConfig(1 * MB),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()

			// Create multipart file.
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile(tt.field, tt.filename)
			if err != nil {
				t.Fatal(err)
			}
			_, err = part.Write(tt.content)
			if err != nil {
				t.Fatal(err)
			}
			writer.Close()

			// Create test request.
			req := httptest.NewRequest("POST", "/", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Parse the multipart form.
			err = req.ParseMultipartForm(32 << 20)
			if err != nil {
				t.Fatal(err)
			}

			// Set the file in validator.
			if req.MultipartForm != nil && req.MultipartForm.File != nil {
				files := req.MultipartForm.File[tt.field]
				if len(files) > 0 {
					v.SetFile(tt.field, files[0])
				}
			}

			file := v.Image(tt.field, tt.config)

			if tt.wantErr {
				if _, ok := v.Errors[tt.field]; !ok {
					t.Errorf("Expected error for field %s", tt.field)
				}
				if file != nil {
					t.Error("Expected nil file when error occurs")
				}
			} else {
				if file == nil {
					t.Error("Expected non-nil file")
				}
				if _, ok := v.Errors[tt.field]; ok {
					t.Errorf("Unexpected error: %v", v.Errors[tt.field])
				}
			}
		})
	}
}

func TestHTTPValidator(t *testing.T) {
	// Create a test form submission.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields.
	_ = writer.WriteField("name", "John Doe")
	_ = writer.WriteField("email", "john@example.com")

	// Add a file.
	part, _ := writer.CreateFormFile("avatar", "test.jpg")
	_, _ = io.Copy(part, strings.NewReader("fake-image-content"))

	writer.Close()

	// Create test request.
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Test the HTTP validator.
	v := NewHTTP(req)

	if v.GetValue("name") != "John Doe" {
		t.Error("Expected name field to be set")
	}

	if v.GetValue("email") != "john@example.com" {
		t.Error("Expected email field to be set")
	}

	if v.GetFile("avatar") == nil {
		t.Error("Expected avatar file to be set")
	}
}

// Helper function to create a test file.
func createTestFile(t *testing.T, name, content string) *os.File {
	t.Helper()

	tmpfile, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile
}

func TestCustomValidations(t *testing.T) {
	tests := []struct {
		name     string
		validate ValidationFunc
		value    string
		wantErr  bool
	}{
		{
			name: "custom validation passes",
			validate: Custom(func(s string) bool {
				return strings.HasPrefix(s, "test")
			}, "Must start with 'test'"),
			value:   "test123",
			wantErr: false,
		},
		{
			name: "custom validation fails",
			validate: Custom(func(s string) bool {
				return strings.HasPrefix(s, "test")
			}, "Must start with 'test'"),
			value:   "invalid123",
			wantErr: true,
		},
		{
			name:     "in string slice passes",
			validate: InStringSlice([]string{"apple", "banana", "orange"}),
			value:    "apple",
			wantErr:  false,
		},
		{
			name:     "in string slice fails",
			validate: InStringSlice([]string{"apple", "banana", "orange"}),
			value:    "grape",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			field := "test_field"
			v.SetValue(field, tt.value)
			v.String(field, tt.validate)

			if tt.wantErr {
				if _, ok := v.Errors[field]; !ok {
					t.Errorf("Expected validation error for field %s", field)
				}
			} else {
				if _, ok := v.Errors[field]; ok {
					t.Errorf("Unexpected validation error for field %s", field)
				}
			}
		})
	}
}
