package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGeneratePreAuthKey_NonEphemeral verifies key generation with no TTL.
func TestGeneratePreAuthKey_NonEphemeral(t *testing.T) {
	k, err := GeneratePreAuthKey(false, 0)
	if err != nil {
		t.Fatalf("GeneratePreAuthKey: %v", err)
	}
	if k.ID == "" {
		t.Error("ID should not be empty")
	}
	if k.Secret == "" {
		t.Error("Secret should not be empty")
	}
	if k.Ephemeral {
		t.Error("Ephemeral should be false")
	}
	if !k.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be zero when no TTL")
	}
	if k.Used {
		t.Error("Used should be false for new key")
	}
}

// TestGeneratePreAuthKey_Ephemeral verifies ephemeral flag and TTL are set.
func TestGeneratePreAuthKey_Ephemeral(t *testing.T) {
	k, err := GeneratePreAuthKey(true, time.Hour)
	if err != nil {
		t.Fatalf("GeneratePreAuthKey: %v", err)
	}
	if !k.Ephemeral {
		t.Error("Ephemeral should be true")
	}
	if k.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set when TTL > 0")
	}
	if time.Until(k.ExpiresAt) < 50*time.Minute {
		t.Error("ExpiresAt should be ~1 hour from now")
	}
}

// TestGeneratePreAuthKey_Unique verifies two keys have different secrets.
func TestGeneratePreAuthKey_Unique(t *testing.T) {
	k1, _ := GeneratePreAuthKey(false, 0)
	k2, _ := GeneratePreAuthKey(false, 0)
	if k1.Secret == k2.Secret {
		t.Error("two generated keys should have different secrets")
	}
	if k1.ID == k2.ID {
		t.Error("two generated keys should have different IDs")
	}
}

// TestPreAuthKey_IsValid_Fresh verifies a fresh non-ephemeral key is valid.
func TestPreAuthKey_IsValid_Fresh(t *testing.T) {
	k, _ := GeneratePreAuthKey(false, 0)
	if !k.IsValid() {
		t.Error("fresh non-ephemeral key should be valid")
	}
}

// TestPreAuthKey_IsValid_Ephemeral_Unused verifies unused ephemeral key is valid.
func TestPreAuthKey_IsValid_Ephemeral_Unused(t *testing.T) {
	k, _ := GeneratePreAuthKey(true, time.Hour)
	if !k.IsValid() {
		t.Error("unused ephemeral key should be valid")
	}
}

// TestPreAuthKey_IsValid_Ephemeral_Used verifies used ephemeral key is invalid.
func TestPreAuthKey_IsValid_Ephemeral_Used(t *testing.T) {
	k, _ := GeneratePreAuthKey(true, time.Hour)
	k.Used = true
	if k.IsValid() {
		t.Error("used ephemeral key should be invalid")
	}
}

// TestPreAuthKey_IsValid_NonEphemeral_Used verifies used non-ephemeral key is still valid.
func TestPreAuthKey_IsValid_NonEphemeral_Used(t *testing.T) {
	k, _ := GeneratePreAuthKey(false, time.Hour)
	k.Used = true
	if !k.IsValid() {
		t.Error("used non-ephemeral key should still be valid")
	}
}

// TestPreAuthKey_IsValid_Expired verifies an expired key is invalid.
func TestPreAuthKey_IsValid_Expired(t *testing.T) {
	k, _ := GeneratePreAuthKey(false, time.Hour)
	k.ExpiresAt = time.Now().Add(-time.Minute) // in the past
	if k.IsValid() {
		t.Error("expired key should be invalid")
	}
}

// TestPreAuthKey_IsValid_FutureExpiry verifies a non-expired key is valid.
func TestPreAuthKey_IsValid_FutureExpiry(t *testing.T) {
	k, _ := GeneratePreAuthKey(false, time.Hour)
	k.ExpiresAt = time.Now().Add(time.Hour)
	if !k.IsValid() {
		t.Error("key with future expiry should be valid")
	}
}

// --- KeyStore tests ---

func newTestKeyStore(t *testing.T) *KeyStore {
	t.Helper()
	ks, err := NewKeyStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return ks
}

// TestNewKeyStore_CreatesDir verifies NewKeyStore creates a nested directory.
func TestNewKeyStore_CreatesDir(t *testing.T) {
	dir := t.TempDir() + "/sub/keys"
	ks, err := NewKeyStore(dir)
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	if ks == nil {
		t.Fatal("NewKeyStore returned nil")
	}
}

// TestKeyStore_SaveLoad verifies Save and Load are inverses.
func TestKeyStore_SaveLoad(t *testing.T) {
	ks := newTestKeyStore(t)
	k, _ := GeneratePreAuthKey(false, 0)

	if err := ks.Save(k); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := ks.Load(k.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != k.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, k.ID)
	}
	if got.Secret != k.Secret {
		t.Errorf("Secret mismatch")
	}
	if got.Ephemeral != k.Ephemeral {
		t.Errorf("Ephemeral mismatch")
	}
}

// TestKeyStore_Load_NotFound verifies Load returns an error for missing IDs.
func TestKeyStore_Load_NotFound(t *testing.T) {
	ks := newTestKeyStore(t)
	if _, err := ks.Load("nonexistent-id"); err == nil {
		t.Fatal("expected error loading nonexistent key")
	}
}

// TestKeyStore_FindBySecret_Found verifies FindBySecret returns the key with matching secret.
func TestKeyStore_FindBySecret_Found(t *testing.T) {
	ks := newTestKeyStore(t)
	k, _ := GeneratePreAuthKey(false, 0)
	ks.Save(k)

	got, err := ks.FindBySecret(k.Secret)
	if err != nil {
		t.Fatalf("FindBySecret: %v", err)
	}
	if got.ID != k.ID {
		t.Errorf("FindBySecret returned wrong key: got %s, want %s", got.ID, k.ID)
	}
}

// TestKeyStore_FindBySecret_NotFound verifies FindBySecret returns an error for unknown secrets.
func TestKeyStore_FindBySecret_NotFound(t *testing.T) {
	ks := newTestKeyStore(t)
	if _, err := ks.FindBySecret("unknown-secret"); err == nil {
		t.Fatal("expected error for unknown secret")
	}
}

// TestKeyStore_List_Empty verifies List returns an empty slice from a fresh store.
func TestKeyStore_List_Empty(t *testing.T) {
	ks := newTestKeyStore(t)
	keys, err := ks.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

// TestKeyStore_List_Multiple verifies List returns all saved keys.
func TestKeyStore_List_Multiple(t *testing.T) {
	ks := newTestKeyStore(t)
	for i := 0; i < 3; i++ {
		k, _ := GeneratePreAuthKey(false, 0)
		if err := ks.Save(k); err != nil {
			t.Fatal(err)
		}
	}
	keys, err := ks.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

// TestKeyStore_Delete verifies that Delete removes a key from the store.
func TestKeyStore_Delete(t *testing.T) {
	ks := newTestKeyStore(t)
	k, _ := GeneratePreAuthKey(false, 0)
	ks.Save(k)

	if err := ks.Delete(k.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := ks.Load(k.ID); err == nil {
		t.Fatal("expected error loading deleted key")
	}
}

// TestKeyStore_Delete_NotFound verifies that deleting a non-existent key returns an error.
func TestKeyStore_Delete_NotFound(t *testing.T) {
	ks := newTestKeyStore(t)
	if err := ks.Delete("nonexistent"); err == nil {
		t.Fatal("expected error deleting nonexistent key")
	}
}

// TestKeyStore_SaveLoad_Ephemeral verifies ephemeral and expiry fields survive the round-trip.
func TestKeyStore_SaveLoad_Ephemeral(t *testing.T) {
	ks := newTestKeyStore(t)
	k, _ := GeneratePreAuthKey(true, 24*time.Hour)
	ks.Save(k)

	got, err := ks.Load(k.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.Ephemeral {
		t.Error("Ephemeral should be preserved after round-trip")
	}
	if got.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be preserved after round-trip")
	}
}

// TestNewKeyStore_MkdirAllFails verifies NewKeyStore returns an error when the
// directory cannot be created (a regular file exists at the target path).
func TestNewKeyStore_MkdirAllFails(t *testing.T) {
	base := t.TempDir()
	// Create a regular file where the key store directory should be.
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	// Attempt to create a key store nested inside the file — must fail.
	_, err := NewKeyStore(filepath.Join(blocker, "keys"))
	if err == nil {
		t.Fatal("expected error when MkdirAll is blocked by a file")
	}
}

// TestKeyStore_FindBySecret_SkipsDir verifies FindBySecret ignores subdirectories
// inside the key store directory (covers the e.IsDir() → continue branch).
func TestKeyStore_FindBySecret_SkipsDir(t *testing.T) {
	ks := newTestKeyStore(t)
	// Create a subdirectory inside the store — directory entries must be skipped.
	subdir := filepath.Join(ks.dir, "subdir")
	if err := os.MkdirAll(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	// With no keys saved, FindBySecret should return not-found (not a panic/crash).
	_, err := ks.FindBySecret("no-such-secret")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

// TestKeyStore_List_SkipsNonJSON verifies List ignores non-.json files and
// subdirectories (covers both IsDir and non-".json" skip branches).
func TestKeyStore_List_SkipsNonJSON(t *testing.T) {
	ks := newTestKeyStore(t)
	// Write a plain text file — should be silently ignored.
	if err := os.WriteFile(filepath.Join(ks.dir, "readme.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	// Create a subdirectory — should also be silently ignored.
	if err := os.MkdirAll(filepath.Join(ks.dir, "subdir"), 0700); err != nil {
		t.Fatal(err)
	}
	keys, err := ks.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys (non-.json and dir entries skipped), got %d", len(keys))
	}
}

// TestKeyStore_Save_OpenFileFails verifies Save returns an error when the file
// cannot be opened for writing (covers the os.OpenFile error path).
func TestKeyStore_Save_OpenFileFails(t *testing.T) {
	ks := newTestKeyStore(t)
	k, _ := GeneratePreAuthKey(false, 0)
	// Block the target path by creating a directory where the file should be.
	blockPath := filepath.Join(ks.dir, k.ID+".json")
	if err := os.MkdirAll(blockPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := ks.Save(k); err == nil {
		t.Fatal("expected error when OpenFile is blocked by directory")
	}
}

// TestKeyStore_Load_BadJSON verifies Load returns an error when the key file
// contains invalid JSON (covers the readJSON error path).
func TestKeyStore_Load_BadJSON(t *testing.T) {
	ks := newTestKeyStore(t)
	// Write a file with invalid JSON content.
	if err := os.WriteFile(filepath.Join(ks.dir, "badkey.json"), []byte("{{{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ks.Load("badkey"); err == nil {
		t.Fatal("expected error for corrupted JSON key file")
	}
}

// TestKeyStore_FindBySecret_ReadDirFails verifies FindBySecret returns an error
// when the store directory cannot be read (covers the os.ReadDir error path).
func TestKeyStore_FindBySecret_ReadDirFails(t *testing.T) {
	ks := newTestKeyStore(t)
	// Remove read permission from the directory.
	if err := os.Chmod(ks.dir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(ks.dir, 0700) // restore for cleanup
	_, err := ks.FindBySecret("secret")
	if err == nil {
		t.Fatal("expected error when ReadDir fails on permission denied")
	}
}

// TestKeyStore_List_ReadDirFails verifies List returns an error when the store
// directory cannot be read (covers the os.ReadDir error path).
func TestKeyStore_List_ReadDirFails(t *testing.T) {
	ks := newTestKeyStore(t)
	// Remove read permission from the directory.
	if err := os.Chmod(ks.dir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(ks.dir, 0700) // restore for cleanup
	_, err := ks.List()
	if err == nil {
		t.Fatal("expected error when ReadDir fails on permission denied")
	}
}
