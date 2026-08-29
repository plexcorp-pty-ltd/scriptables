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

var nonKeyFiles = map[string]bool{
	"authorized_keys":  true,
	"authorized_keys2": true,
	"config":           true,
	"known_hosts":      true,
	"known_hosts2":     true,
	"environment":      true,
	"rc":               true,
}

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

type LocalKey struct {
	Name      string // file name of the private key, e.g. "id_ed25519"
	Path      string // full path to the private key
	Signer    ssh.Signer
	PublicKey string // authorized_keys formatted public key
	Encrypted bool   // true when the private key is passphrase protected
}

var preferredOrder = []string{"id_ed25519", "id_ecdsa", "id_rsa", "id_dsa", "identity"}

func rank(name string) int {
	for i, preferred := range preferredOrder {
		if name == preferred {
			return i
		}
	}
	return len(preferredOrder)
}

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

func dialTimeout() time.Duration {
	if raw := os.Getenv("SCRIPTABLES_SSH_TIMEOUT"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}

	return 15 * time.Second
}

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
		Timeout: dialTimeout(),
	}

	return Dial("tcp", addr, config)
}
