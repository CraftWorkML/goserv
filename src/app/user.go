package app

// S3Image represents an image stored in an S3 bucket.
type S3Image struct {
	// The key (object name) of the image in the S3 bucket.
	Key string `json:"key"`

	// The URL of the image in the S3 bucket.
	URL string `json:"url"`

	// The MIME type of the image (e.g., "image/jpeg", "image/png").
	ContentType string `json:"content_type"`

	// The size of the image in bytes.
	Size int64 `json:"size"`

	// Additional metadata or properties related to the image.
	Metadata map[string]string `json:"metadata"`
}

// User represents a user with a one-to-many relationship to S3Images.
type User struct {
	// Unique user ID in your application.
	ID string `json:"id"`

	// User's preferred username, often used for display.
	Username string `json:"username"`

	Picture string `json:"picture"`

	// User's email address.
	Email string `json:"email"`

	// User's display name.
	Name string `json:"name"`

	// A slice of S3Image objects representing the images associated with the user.
	Images []S3Image `json:"images"`
}
