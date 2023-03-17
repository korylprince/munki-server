package main

// Config configures munki-server
type Config struct {
	// Webroot is the path to the root directory served by the file and webdav servers
	WebRoot string `required:"true"`
	// ManifestRoot is the relative path to the manifest directory from WebRoot. Should have a leading and trailing slash (/)
	ManifestRoot    string `required:"true"`
	AssignmentsPath string `required:"true"`
	// WebDAVPrefix is the path where the WebDAV share is mounted. Should have a leading and trailing slash (/)
	WebDAVPrefix string `default:"/edit/"`

	Username string `default:"webdav"`
	Password string `required:"true"`

	ProxyHeaders bool   `default:"false"`
	ListenAddr   string `default:":80"`
}
