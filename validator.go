package form_validator

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Add new types and constants for file validation.
type FileValidationConfig struct {
	MaxSize      int64    // maximum file size in bytes.
	AllowedTypes []string // allowed MIME types.
	AllowedExts  []string // allowed file extensions.
}

// Common MIME types for images.
const (
	MimeJPEG = "image/jpeg"
	MimePNG  = "image/png"
	MimeGIF  = "image/gif"
	MimeWEBP = "image/webp"
)

// Default image formats.
var DefaultImageFormats = []string{"jpg", "jpeg", "png", "gif", "webp"}

// Validator holds the validation errors and form values.
type Validator struct {
	Errors map[string]string
	values map[string]string
	files  map[string]*multipart.FileHeader
}

// Common file size constants.
const (
	KB = 1024
	MB = 1024 * KB
)

// ValidationFunc represents a validation function.
type ValidationFunc func(field, value string) (bool, string)

// New creates a new validator instance.
func New() *Validator {
	return &Validator{
		Errors: make(map[string]string),
		values: make(map[string]string),
		files:  make(map[string]*multipart.FileHeader),
	}
}

// SetValue sets a form value.
func (v *Validator) SetValue(field, value string) {
	v.values[field] = value
}

// GetValue gets a form value.
func (v *Validator) GetValue(field string) string {
	return v.values[field]
}

// Add method to set file.
func (v *Validator) SetFile(field string, file *multipart.FileHeader) {
	v.files[field] = file
}

// Add method to get file.
func (v *Validator) GetFile(field string) *multipart.FileHeader {
	return v.files[field]
}

// ImageConfig creates a standard image validation configuration.
func ImageConfig(maxSize int64, formats ...string) FileValidationConfig {
	if len(formats) == 0 {
		formats = DefaultImageFormats
	}

	mimeTypes := make([]string, 0)
	extensions := make([]string, 0)

	for _, format := range formats {
		switch strings.ToLower(format) {
		case "jpg", "jpeg":
			mimeTypes = append(mimeTypes, MimeJPEG)
			extensions = append(extensions, ".jpg", ".jpeg")
		case "png":
			mimeTypes = append(mimeTypes, MimePNG)
			extensions = append(extensions, ".png")
		case "gif":
			mimeTypes = append(mimeTypes, MimeGIF)
			extensions = append(extensions, ".gif")
		case "webp":
			mimeTypes = append(mimeTypes, MimeWEBP)
			extensions = append(extensions, ".webp")
		}
	}

	return FileValidationConfig{
		MaxSize:      maxSize,
		AllowedTypes: mimeTypes,
		AllowedExts:  extensions,
	}
}

// Image validates an image file field.
func (v *Validator) Image(field string, config FileValidationConfig) *multipart.FileHeader {
	file := v.files[field]
	if file == nil {
		v.Errors[field] = "No file was uploaded"
		return nil
	}

	// Validate file size.
	if config.MaxSize > 0 && file.Size > config.MaxSize {
		v.Errors[field] = fmt.Sprintf("File size exceeds maximum limit of %d bytes", config.MaxSize)
		return nil
	}

	// Validate file extension.
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if len(config.AllowedExts) > 0 {
		validExt := false
		for _, allowedExt := range config.AllowedExts {
			if strings.ToLower(allowedExt) == ext {
				validExt = true
				break
			}
		}

		if !validExt {
			v.Errors[field] = fmt.Sprintf("Invalid file extension. Allowed: %s", strings.Join(config.AllowedExts, ", "))
			return nil
		}
	}

	// Validate MIME type.
	f, err := file.Open()
	if err != nil {
		v.Errors[field] = "Could not process file"
		return nil
	}
	defer f.Close()

	// Read first 512 bytes for MIME type detection.
	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil && err != io.EOF {
		v.Errors[field] = "Could not read file content"
		return nil
	}

	detectedType := http.DetectContentType(buffer)
	if len(config.AllowedTypes) > 0 {
		validType := false
		for _, allowedType := range config.AllowedTypes {
			if strings.HasPrefix(detectedType, allowedType) {
				validType = true
				break
			}
		}

		if !validType {
			v.Errors[field] = fmt.Sprintf("Invalid file type. Allowed: %s", strings.Join(config.AllowedTypes, ", "))
			return nil
		}
	}

	return file
}

// String validates a string field with the given validation functions
func (v *Validator) String(field string, validations ...ValidationFunc) string {
	value := v.GetValue(field)

	for _, validation := range validations {
		if ok, message := validation(field, value); !ok {
			v.Errors[field] = message
			break
		}
	}

	return value
}

// Int validates and returns an integer field.
func (v *Validator) Int(field string, validations ...ValidationFunc) int64 {
	value := v.GetValue(field)

	for _, validation := range validations {
		if ok, message := validation(field, value); !ok {
			v.Errors[field] = message
			break
		}
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		v.Errors[field] = "This field must be a valid integer"
		return 0
	}

	return intValue
}

// Check adds an error if the condition is false.
func (v *Validator) Check(ok bool, field, message string) {
	if !ok {
		v.Errors[field] = message
	}
}

// Validate returns true if there are no errors.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// Predefined validation functions.

// Required validates that a field is not empty
func Required(field, value string) (bool, string) {
	if strings.TrimSpace(value) == "" {
		return false, "This field is required"
	}

	return true, ""
}

// MinLength creates a validation function for minimum length
func MinLength(min int) ValidationFunc {
	return func(field, value string) (bool, string) {
		if utf8.RuneCountInString(value) < min {
			return false, fmt.Sprintf("This field must be at least %d characters long", min)
		}

		return true, ""
	}
}

// MaxLength creates a validation function for maximum length
func MaxLength(max int) ValidationFunc {
	return func(field, value string) (bool, string) {
		if utf8.RuneCountInString(value) > max {
			return false, fmt.Sprintf("This field must not exceed %d characters", max)
		}

		return true, ""
	}
}

// Email validates email format
func Email(field, value string) (bool, string) {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(pattern)

	if !regex.MatchString(value) {
		return false, "Please enter a valid email address"
	}

	return true, ""
}

// Matches creates a validation function for regex pattern matching
func Matches(pattern string, message string) ValidationFunc {
	return func(field, value string) (bool, string) {
		regex := regexp.MustCompile(pattern)

		if !regex.MatchString(value) {
			return false, message
		}

		return true, ""
	}
}

// Boolean validates that a value is "true" or "false"
func Boolean(field, value string) (bool, string) {
	value = strings.TrimSpace(strings.ToLower(value))

	_, err := strconv.ParseBool(value)
	if err != nil {
		return false, "This field must be true or false"
	}

	return true, ""
}

// IntRange creates a validation function for integer range.
func IntRange(min, max int) ValidationFunc {
	return func(field, value string) (bool, string) {
		// Add proper int conversion and range check here
		return true, ""
	}
}

// InStringSlice creates a validation function that checks if a value exists in a slice.
func InStringSlice(slice []string) ValidationFunc {
	return func(field, value string) (bool, string) {
		for _, item := range slice {
			if item == value {
				return true, ""
			}
		}

		return false, "This value is not in the allowed list"
	}
}

// Custom creates a validation function from a custom check.
func Custom(check func(string) bool, message string) ValidationFunc {
	return func(field, value string) (bool, string) {
		if !check(value) {
			return false, message
		}
		return true, ""
	}
}

// HTTPValidator extends Validator to work with http.Request.
type HTTPValidator struct {
	*Validator
	request *http.Request
}

// NewHTTP creates a new HTTP validator.
func NewHTTP(r *http.Request) *HTTPValidator {
	v := &HTTPValidator{
		Validator: New(),
		request:   r,
	}

	// Check if it's a multipart form.
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory.
		if err == nil {
			// Load files
			if r.MultipartForm != nil && r.MultipartForm.File != nil {
				for field, files := range r.MultipartForm.File {
					if len(files) > 0 {
						v.SetFile(field, files[0])
					}
				}
			}
		}
	}

	// Parse regular form values.
	r.ParseForm()
	for key, values := range r.Form {
		if len(values) > 0 {
			v.SetValue(key, values[0])
		}
	}

	return v
}
