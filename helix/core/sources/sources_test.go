package sources

import "testing"

func TestAllProtocolsStreamOrRange(t *testing.T) {
	for p, caps := range capabilities {
		if !caps.RangeRead && !caps.Streaming && p != Email {
			t.Errorf("%s supports neither range nor streaming", p)
		}
	}
}

func TestInsecureProtocolsFlagged(t *testing.T) {
	if Secure(FTP) || Secure(HTTP) || Secure(SMB) || Secure(OPDS) || Secure(LocalFolder) {
		t.Error("optionally cleartext/local protocols must not be marked inherently secure")
	}
	if !Secure(SFTP) || !Secure(FTPS) || !Secure(GoogleDrive) {
		t.Error("TLS/SSH-required protocols must be secure")
	}
}

func TestSyncCapableBackends(t *testing.T) {
	// WebDAV must be usable as a sync destination (docs/25.1)
	if !Capabilities(WebDAV).Write {
		t.Error("WebDAV must support write (sync destination)")
	}
}
