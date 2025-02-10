# Form Validator

A simple yet powerful form validation package for Go web applications.

## Features

- String validation (required, email, length checks, in slice, matches ...)
- Integer validation
- Boolean validation
- File upload validation (size, type, extension)
- Custom validation rules
- HTTP form integration
- Built-in validation for common image formats

## Installation

```go
go get github.com/yourusername/form_validator
```

## Basic Usage

```go
func handleForm(w http.ResponseWriter, r *http.Request) {
    // Create validator from HTTP request.
    v := form_validator.NewHTTP(r)
    
    // Validate form fields.
    username := v.String("username", Required, MinLength(3))
    email := v.String("email", Required, Email)
    age := v.Int("age")
    
    // Validate file upload.
    config := form_validator.ImageConfig(1 * form_validator.MB)
    avatar := v.Image("avatar", config)
    
    if !v.Valid() {
        // Handle validation errors.
        return
    }
    
    // Process valid form...
    fmt.Println(username, email, age, avatar)
}
```
### Custom Validation Rules

```go
// Create custom validation function.
isValidUsername := Custom(
    func(value string) bool {
        // Add your custom validation logic.
        return len(value) >= 3 && !strings.Contains(value, " ")
    },
    "Username must be at least 3 characters and contain no spaces",
)

// Use custom validation.
username := v.String("username", Required, isValidUsername)
```

### File Upload Validation

```go
func handleImageUpload(w http.ResponseWriter, r *http.Request) {
    v := form_validator.NewHTTP(r)
    
    // Configure image validation
    config := form_validator.ImageConfig(
        5 * form_validator.MB,  // Max size: 5MB.
        "jpg", "png", "webp",  // Allowed formats.
    )
    
    // Validate the uploaded image.
    file := v.Image("profile_picture", config)
    if !v.Valid() {
        // Handle validation errors.
        fmt.Println(v.Errors["profile_picture"])
        return
    }
    
    // Process valid file...
}
```

## Best Practices

1. Always check `v.Valid()` before processing form data
2. Use appropriate validation functions for each field type
3. Set reasonable file size limits for uploads
4. Validate file types and extensions for security
5. Handle validation errors appropriately in your application

## License

This project is licensed under the MIT License - see the LICENSE file for details.

