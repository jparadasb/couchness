package models

import "testing"

func TestValidTelegramRole(t *testing.T) {
	valid := []string{TelegramRoleOwner, TelegramRoleAdmin, TelegramRoleUser, TelegramRoleViewer}
	for _, role := range valid {
		if !ValidTelegramRole(role) {
			t.Errorf("expected %q to be valid", role)
		}
	}
	if ValidTelegramRole("guest") {
		t.Error("expected unknown role to be invalid")
	}
}
