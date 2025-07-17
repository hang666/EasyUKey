package test

import (
	"testing"

	"github.com/hang666/EasyUKey/sdk"
	"github.com/hang666/EasyUKey/sdk/request"
)

func TestAuth(t *testing.T) {
	client := sdk.NewClient("http://localhost:8888", "admin-key-easyukey-2024")

	authData, err := client.StartAuth(
		"testuser",
		&request.AuthRequest{
			Challenge: "123456",
			Timeout:   600,
			UserID:    "testuser",
			Action:    "",
			Message:   "test",
		},
	)
	t.Logf("authData: %v", authData)
	if err != nil {
		t.Fatalf("StartAuth failed: %v", err)
	}
}
