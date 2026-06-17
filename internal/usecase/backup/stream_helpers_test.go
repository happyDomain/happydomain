package backup_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/internal/usecase/backup"
	happydns "git.happydns.org/happyDomain/model"
)

// dumpAll runs the streaming full backup and decodes it back.
func dumpAll(t *testing.T, uc *backup.Usecase) happydns.Backup {
	t.Helper()

	var buf bytes.Buffer
	if err := uc.WriteBackup(&buf); err != nil {
		t.Fatalf("WriteBackup: %v", err)
	}
	var ret happydns.Backup
	if err := json.Unmarshal(buf.Bytes(), &ret); err != nil {
		t.Fatalf("WriteBackup produced invalid JSON: %v", err)
	}
	return ret
}

// dumpUser runs the streaming user export and decodes it back.
func dumpUser(t *testing.T, uc *backup.Usecase, user *happydns.User) happydns.Backup {
	t.Helper()

	var buf bytes.Buffer
	if err := uc.WriteBackupUser(&buf, user); err != nil {
		t.Fatalf("WriteBackupUser: %v", err)
	}
	var ret happydns.Backup
	if err := json.Unmarshal(buf.Bytes(), &ret); err != nil {
		t.Fatalf("WriteBackupUser produced invalid JSON: %v", err)
	}
	return ret
}
