// Package sources models the Source Abstraction Layer capabilities
// (docs/25): one interface, 20+ protocol backends, decoupled from sync.
package sources

// Capability set a backend can declare.
type Caps struct {
	RangeRead bool
	Watch     bool
	Write     bool
	Streaming bool
}

type Protocol string

const (
	OPDS        Protocol = "opds"
	Calibre     Protocol = "calibre"
	WebDAV      Protocol = "webdav"
	FTP         Protocol = "ftp"
	FTPS        Protocol = "ftps"
	SFTP        Protocol = "sftp"
	SMB         Protocol = "smb"
	NFS         Protocol = "nfs"
	HTTP        Protocol = "http"
	S3          Protocol = "s3"
	GoogleDrive Protocol = "google_drive"
	Dropbox     Protocol = "dropbox"
	OneDrive    Protocol = "onedrive"
	Nextcloud   Protocol = "nextcloud"
	Email       Protocol = "email_ingest"
	RSS         Protocol = "rss"
	LocalFolder Protocol = "local_folder"
)

// capabilities is the audited capability table (docs/25.2).
var capabilities = map[Protocol]Caps{
	OPDS:        {RangeRead: true, Streaming: true},
	Calibre:     {RangeRead: true, Streaming: true},
	WebDAV:      {RangeRead: true, Write: true, Streaming: true},
	FTP:         {RangeRead: true, Streaming: true},
	FTPS:        {RangeRead: true, Streaming: true},
	SFTP:        {RangeRead: true, Write: true, Streaming: true},
	SMB:         {RangeRead: true, Watch: true, Write: true, Streaming: true},
	NFS:         {RangeRead: true, Write: true, Streaming: true},
	HTTP:        {RangeRead: true, Streaming: true},
	S3:          {RangeRead: true, Write: true, Streaming: true},
	GoogleDrive: {RangeRead: true, Watch: true, Write: true, Streaming: true},
	Dropbox:     {RangeRead: true, Watch: true, Write: true, Streaming: true},
	OneDrive:    {RangeRead: true, Watch: true, Write: true, Streaming: true},
	Nextcloud:   {RangeRead: true, Write: true, Streaming: true},
	Email:       {},
	RSS:         {Streaming: true},
	LocalFolder: {RangeRead: true, Watch: true, Write: true, Streaming: true},
}

func Capabilities(p Protocol) Caps { return capabilities[p] }

// Secure reports whether the protocol requires transport encryption by
// construction. SMB, OPDS, WebDAV, S3 and self-hosted protocols can also run
// without encryption and therefore must not be treated as inherently secure.
func Secure(p Protocol) bool {
	switch p {
	case FTPS, SFTP, GoogleDrive, Dropbox, OneDrive:
		return true
	default:
		return false
	}
}
