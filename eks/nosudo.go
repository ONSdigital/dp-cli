package eks

// ResolveNoSudoMode decides whether to use the no-sudo tunnel path.
//
// No-sudo mode is the DEFAULT. It works on any device regardless of whether the
// user has admin/root rights, keeps TLS fully validated via kubeconfig
// tls-server-name, and needs neither sudo nor socat. Engineers who specifically
// want the legacy socat/hosts/loopback path (for example to present the tunnel
// on the real hostname:443 for a tool that cannot use kubeconfig) can opt in
// with --legacy-sudo.
func ResolveNoSudoMode(legacySudoFlag bool) bool {
	return !legacySudoFlag
}
