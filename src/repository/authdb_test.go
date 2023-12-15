package repository

import (
	"testing"

	cfg "goserv/src/configuration"
)

func TestInMemoryDB(t *testing.T) {
	// Create a configuration for the database (can be a mock configuration)
	config := &cfg.Properties{}
	db, err := NewAuthDataBase(config)

	// Check if creating the database instance was successful
	if err != nil {
		t.Fatalf("Error creating AuthDB instance: %v", err)
	}

	// Test Connect method
	t.Run("Connect", func(t *testing.T) {
		result := db.Connect()
		if !result {
			t.Error("Connect() returned false, expected true")
		}
	})

	// Test UploadUser method
	t.Run("UploadUser", func(t *testing.T) {
		accessToken := "someAccessToken"
		refreshToken := "someRefreshToken"

		err := db.UploadUser(accessToken, refreshToken)
		if err != nil {
			t.Errorf("UploadUser() returned an error: %v", err)
		}

		// Check if the user is present in the database
		if !db.VerifyUser(accessToken) {
			t.Error("Uploaded user not found in the database")
		}
	})

	// Test VerifyUser method
	t.Run("VerifyUser", func(t *testing.T) {
		accessToken := "someAccessToken"
		// Check if a non-existent user is not verified
		if !db.VerifyUser(accessToken) {
			t.Error("VerifyUser() returned true for a non-existent user, expected false")
		}
	})
}
