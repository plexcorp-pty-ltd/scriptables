package sshclient

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Files that live in ~/.ssh but are never private keys.
var nonKeyFiles = map[string]bool{
	"authorized_keys":  true,
	"authorized_keys2": true,
	"config":           true,
	"known_hosts":      true,
	"known_hosts2":     true,
	"environment":      true,
	"rc":               true,
}

// SshDir returns the directory Scriptables reads SSH keys from. Scriptables runs
// natively on your machine, so it simply re-uses whatever keys you already have.
// Set SCRIPTABLES_SSH_DIR to point at a different directory.
func SshDir() string {
	if dir := os.Getenv("SCRIPTABLES_SSH_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".ssh"
	}

	return filepath.Join(home, ".ssh")
}

func isCandidateKeyFile(name string) bool {
	if strings.HasPrefix(name, ".") || nonKeyFiles[name] {
		return false
	}

	switch filepath.Ext(name) {
	case ".pub", ".old", ".bak", ".ppk", ".sock":
		return false
	}

	return true
}

// LocalKey is a usable key pair found in the SSH directory.
type LocalKey struct {
	Name      string // file name of the private key, e.g. "id_ed25519"
	Path      string // full path to the private key
	Signer    ssh.Signer
	PublicKey string // authorized_keys formatted public key
	Encrypted bool   // true when the private key is passphrase protected
}

// preferredOrder puts the conventional key names first. SSH servers cap the
// number of authentication attempts (OpenSSH allows 6 by default), so the keys
// most likely to be the right one are offered first.
var preferredOrder = []string{"id_ed25519", "id_ecdsa", "id_rsa", "id_dsa", "identity"}

func rank(name string) int {
	for i, preferred := range preferredOrder {
		if name == preferred {
			return i
		}
	}
	return len(preferredOrder)
}

// selectedKeyNames honours SCRIPTABLES_SSH_KEYS, a comma separated list of key
// file names. Set it when you have more keys than the server will let us try.
func selectedKeyNames() map[string]bool {
	raw := os.Getenv("SCRIPTABLES_SSH_KEYS")
	if raw == "" {
		return nil
	}

	selected := map[string]bool{}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			selected[filepath.Base(name)] = true
		}
	}

	if len(selected) == 0 {
		return nil
	}

	return selected
}

// LocalKeys lists every private key in the SSH directory that can be loaded
// without a passphrase. Passphrase protected keys are reported with Encrypted
// set so the UI can explain why they are unavailable - use an ssh-agent for those.
func LocalKeys() ([]LocalKey, error) {
	dir := SshDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	selected := selectedKeyNames()

	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !isCandidateKeyFile(entry.Name()) {
			continue
		}
		if selected != nil && !selected[entry.Name()] {
			continue
		}
		names = append(names, entry.Name())
	}

	sort.Slice(names, func(i, j int) bool {
		if ri, rj := rank(names[i]), rank(names[j]); ri != rj {
			return ri < rj
		}
		return names[i] < names[j]
	})

	keys := []LocalKey{}
	for _, name := range names {
		path := filepath.Join(dir, name)
		contents, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		signer, err := ssh.ParsePrivateKey(contents)
		if err != nil {
			var missing *ssh.PassphraseMissingError
			if errors.As(err, &missing) {
				keys = append(keys, LocalKey{Name: name, Path: path, Encrypted: true})
			}
			continue
		}

		keys = append(keys, LocalKey{
			Name:      name,
			Path:      path,
			Signer:    signer,
			PublicKey: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))),
		})
	}

	return keys, nil
}

// agentSigners returns the signers held by a running ssh-agent, if there is one.
// This is how passphrase protected keys are supported.
func agentSigners() []ssh.Signer {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}

	signers, err := agent.NewClient(conn).Signers()
	if err != nil {
		return nil
	}

	return signers
}

// LocalSigners returns every key we can authenticate with: the ssh-agent keys
// first (they cover passphrase protected keys) followed by the on disk keys.
func LocalSigners() ([]ssh.Signer, error) {
	signers := agentSigners()

	keys, err := LocalKeys()
	if err != nil && len(signers) == 0 {
		return nil, err
	}

	for _, key := range keys {
		if key.Signer != nil {
			signers = append(signers, key.Signer)
		}
	}

	if len(signers) == 0 {
		return nil, errors.New("no usable SSH keys found in " + SshDir() +
			". Generate one with `ssh-keygen -t ed25519`, or add a passphrase protected key to your ssh-agent.")
	}

	return signers, nil
}

// LocalPublicKeys returns the authorized_keys lines for every key we can use.
// These get installed on the servers Scriptables builds.
func LocalPublicKeys() []string {
	seen := map[string]bool{}
	pubKeys := []string{}

	add := func(pub ssh.PublicKey) {
		line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
		if line == "" || seen[line] {
			return
		}
		seen[line] = true
		pubKeys = append(pubKeys, line)
	}

	for _, signer := range agentSigners() {
		add(signer.PublicKey())
	}

	keys, _ := LocalKeys()
	for _, key := range keys {
		if key.Signer != nil {
			add(key.Signer.PublicKey())
		}
	}

	// Passphrase protected keys have no signer, but their .pub file still tells
	// us what to install on the server.
	for _, key := range keys {
		if key.Signer != nil {
			continue
		}
		contents, err := os.ReadFile(key.Path + ".pub")
		if err != nil {
			continue
		}
		pub, _, _, _, err := ssh.ParseAuthorizedKey(contents)
		if err != nil {
			continue
		}
		add(pub)
	}

	return pubKeys
}

// dialTimeout caps how long we wait for a TCP connection and handshake.
// Override with SCRIPTABLES_SSH_TIMEOUT (e.g. "5s", "30s").
func dialTimeout() time.Duration {
	if raw := os.Getenv("SCRIPTABLES_SSH_TIMEOUT"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}

	return 15 * time.Second
}

// DialWithLocalKeys connects to addr, offering every SSH key available on this
// machine. The server picks the one it accepts.
func DialWithLocalKeys(addr, user string) (*Client, error) {
	signers, err := LocalSigners()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		// Without this a wrong or unroutable IP hangs the request forever, which
		// leaves the connection test and firewall pages spinning indefinitely.
		Timeout: dialTimeout(),
	}

	return Dial("tcp", addr, config)
}
