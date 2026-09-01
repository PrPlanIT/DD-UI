package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptossh "golang.org/x/crypto/ssh"
)

// remoteAuth builds the auth method for a repository from whichever credential the
// caller holds. An SSH key is parsed in memory rather than written to /tmp: the file
// only ever existed so the git binary could be pointed at it with GIT_SSH_COMMAND, and
// a private key that is never written cannot be left behind by a crash.
func remoteAuth(repoURL, authToken, sshKey string) (transport.AuthMethod, error) {
	switch {
	case authToken != "":
		// Username is ignored by every forge when the password is a token, but it may
		// not be empty.
		return &githttp.BasicAuth{Username: "token", Password: authToken}, nil

	case sshKey != "":
		key := sshKey
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}
		user := "git"
		if i := strings.Index(repoURL, "@"); i > 0 && !strings.Contains(repoURL, "://") {
			user = repoURL[:i]
		}
		auth, err := gitssh.NewPublicKeys(user, []byte(key), "")
		if err != nil {
			return nil, fmt.Errorf("ssh key: %w", err)
		}
		// Preserves the previous StrictHostKeyChecking=no behaviour. Enabling
		// verification here would reject every host whose key is not already trusted,
		// which for this feature is every host a user has just typed in.
		auth.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()
		return auth, nil
	}
	return nil, nil
}

// listRemoteHeads reports the branch names a remote advertises, without cloning and
// without a git binary — the runtime image has none, so shelling out to `git ls-remote`
// failed there whatever the repository or credentials were.
func listRemoteHeads(repoURL, authToken, sshKey string) ([]string, error) {
	auth, err := remoteAuth(repoURL, authToken, sshKey)
	if err != nil {
		return nil, err
	}

	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	refs, err := rem.List(&git.ListOptions{Auth: auth})
	if err != nil {
		return nil, err
	}

	var heads []string
	for _, r := range refs {
		if r.Name().IsBranch() {
			heads = append(heads, r.Name().Short())
		}
	}
	return heads, nil
}

// remoteErrorMessage turns a go-git transport failure into something a user can act on.
//
// The previous implementation matched substrings of git's stderr, which meant the
// message depended on the wording of whichever git build the image happened to carry.
// go-git reports these as typed errors, so the common cases are matched on identity and
// only the genuinely unrecognised ones fall back to the raw text.
func remoteErrorMessage(err error) string {
	switch {
	case errors.Is(err, transport.ErrAuthenticationRequired):
		return "Authentication required. Provide a token or SSH key for this repository."
	case errors.Is(err, transport.ErrAuthorizationFailed):
		return "Authentication failed. Check your credentials have access to the repository."
	case errors.Is(err, transport.ErrRepositoryNotFound):
		return "Repository not found. Check the URL and access permissions."
	case errors.Is(err, transport.ErrEmptyRemoteRepository):
		// Reachable and authorized, with nothing published yet.
		return ""
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "unable to authenticate"), strings.Contains(msg, "handshake failed"):
		return "SSH authentication failed. Please check your SSH key has access to the repository."
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "Name or service not known"):
		return "Cannot reach repository. Check the URL and network connection."
	case strings.Contains(msg, "connection refused"):
		return "Connection refused. Check if the Git server is running and the port is correct."
	case strings.Contains(msg, "i/o timeout"):
		return "Connection timed out. Check the URL and network connection."
	}
	return fmt.Sprintf("Connection failed: %s", strings.TrimSpace(msg))
}

// branchInHeads reports whether the remote advertises the named branch.
func branchInHeads(heads []string, branch string) bool {
	for _, h := range heads {
		if h == branch {
			return true
		}
	}
	return false
}
