//go:build linux

package detector

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/maximbaz/yubikey-touch-detector/notifier"
)

type sshProxySession struct {
	listener    *net.UnixListener
	publicPath  string
	backendPath string
	publicInfo  os.FileInfo
	connections sync.Map
	failure     chan struct{}
	failOnce    sync.Once
}

// pathChanged reports whether the public socket file was replaced (for example
// because the agent was restarted and re-created it), in which case the proxy
// no longer fronts the live socket and must be reinstalled.
func (session *sshProxySession) pathChanged() bool {
	info, err := os.Lstat(session.publicPath)
	return err != nil || !os.SameFile(info, session.publicInfo)
}

func (session *sshProxySession) fail() {
	session.failOnce.Do(func() {
		close(session.failure)
	})
}

func (session *sshProxySession) closeConnections() {
	session.connections.Range(func(connection, _ interface{}) bool {
		if conn, ok := connection.(net.Conn); ok {
			conn.Close()
		}
		return true
	})
}

// close tears down the proxy. When restore is true (clean shutdown) the
// original agent socket is put back in place; when false (reinstall) it is left
// untouched so a socket the agent re-created underneath us is not clobbered.
func (session *sshProxySession) close(restore bool) {
	ownsPublicPath := !session.pathChanged()
	session.listener.Close()
	session.closeConnections()
	if !restore {
		return
	}
	if !ownsPublicPath {
		log.Warn("Cannot restore original SSH agent socket because the public socket was replaced")
		return
	}
	if err := os.Rename(session.backendPath, session.publicPath); err != nil {
		log.Errorf("Cannot restore original SSH agent socket: %v", err)
	}
}

// installSSHProxy renames the real agent socket to "<path>.original" and binds
// a proxy on the public path, recovering from a stale proxy/backend left behind
// by a previous crash.
func installSSHProxy(publicPath string) (*sshProxySession, error) {
	backendPath := publicPath + ".original"
	_, publicErr := os.Lstat(publicPath)
	_, backendErr := os.Lstat(backendPath)
	publicExists := publicErr == nil
	backendExists := backendErr == nil
	publicLive := publicExists && socketAcceptsConnections(publicPath)
	backendLive := backendExists && socketAcceptsConnections(backendPath)

	if backendLive {
		if publicLive {
			return nil, errors.New("both SSH proxy and backend sockets are active")
		}
		if publicExists {
			if err := os.Remove(publicPath); err != nil {
				return nil, err
			}
		}
	} else {
		if !publicLive {
			if publicExists {
				_ = os.Remove(publicPath)
			}
			if backendExists {
				_ = os.Remove(backendPath)
			}
			return nil, os.ErrNotExist
		}
		if err := os.Rename(publicPath, backendPath); err != nil {
			return nil, err
		}
	}

	address, err := net.ResolveUnixAddr("unix", publicPath)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenUnix("unix", address)
	if err != nil {
		_ = os.Rename(backendPath, publicPath)
		return nil, err
	}
	listener.SetUnlinkOnClose(false)
	if err := os.Chmod(publicPath, 0o600); err != nil {
		listener.Close()
		_ = os.Rename(backendPath, publicPath)
		return nil, err
	}
	publicInfo, err := os.Lstat(publicPath)
	if err != nil {
		listener.Close()
		_ = os.Rename(backendPath, publicPath)
		return nil, err
	}

	return &sshProxySession{
		listener:    listener,
		publicPath:  publicPath,
		backendPath: backendPath,
		publicInfo:  publicInfo,
		failure:     make(chan struct{}),
	}, nil
}

func (session *sshProxySession) handleConnection(proxyConnection *net.UnixConn, requestGPGCheck chan *notifier.Process) {
	session.connections.Store(proxyConnection, true)
	defer session.connections.Delete(proxyConnection)
	defer proxyConnection.Close()

	process := processFromPeer(proxyConnection)
	originalConnection, err := net.Dial("unix", session.backendPath)
	if err != nil {
		log.Error("Cannot establish connection to original SSH socket: ", err)
		session.fail()
		return
	}
	session.connections.Store(originalConnection, true)
	defer session.connections.Delete(originalConnection)
	defer originalConnection.Close()

	go proxyUnixSocket(proxyConnection, originalConnection, process, requestGPGCheck)
	proxyUnixSocket(originalConnection, proxyConnection, process, requestGPGCheck)
}

func (session *sshProxySession) accept(requestGPGCheck chan *notifier.Process) error {
	for {
		connection, err := session.listener.AcceptUnix()
		if err != nil {
			return err
		}
		go session.handleConnection(connection, requestGPGCheck)
	}
}

func resolveSSHAuthSocket() string {
	socketFile := os.Getenv("SSH_AUTH_SOCK")

	if socketFile == "" {
		gpgAgentSocket, err := exec.Command("gpgconf", "--list-dirs", "agent-ssh-socket").CombinedOutput()
		gpgAgentSocketOutput := strings.TrimSpace(string(gpgAgentSocket))
		if err != nil {
			log.Errorf("Cannot find SSH socket using gpgconf, error: %v, stderr: %v", err, gpgAgentSocketOutput)
		} else {
			socketFile = gpgAgentSocketOutput
		}
	}

	if socketFile == "" {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir != "" {
			socketFile = path.Join(runtimeDir, "gnupg/S.gpg-agent.ssh")
		}
	}

	if socketFile == "" {
		log.Error("Cannot watch SSH. $SSH_AUTH_SOCK is not defined, gpgconf --list-dirs agent-ssh-socket didn't help, and $XDG_RUNTIME_DIR is not defined.")
	}
	return socketFile
}

// WatchSSH watches for hints that YubiKey is maybe waiting for a touch on a SSH auth request
func WatchSSH(requestGPGCheck chan *notifier.Process, exits *sync.Map) {
	socketFile := resolveSSHAuthSocket()
	if socketFile == "" {
		return
	}

	exit := make(chan bool)
	exits.Store("detector/ssh", exit)
	defer exits.Delete("detector/ssh")

	lastError := ""
	for {
		session, err := installSSHProxy(socketFile)
		if err != nil {
			if message := err.Error(); message != lastError {
				log.Errorf("Cannot establish SSH proxy: %v", err)
				lastError = message
			}
			select {
			case <-exit:
				exit <- true
				return
			case <-time.After(time.Second):
			}
			continue
		}
		lastError = ""
		log.Debug("SSH watcher is successfully established")

		acceptResult := make(chan error, 1)
		go func() {
			acceptResult <- session.accept(requestGPGCheck)
		}()
		ticker := time.NewTicker(time.Second)
		restart := false
		for !restart {
			select {
			case <-exit:
				ticker.Stop()
				session.close(true)
				exit <- true
				return
			case err := <-acceptResult:
				if !errors.Is(err, net.ErrClosed) {
					log.Error("SSH proxy stopped accepting connections: ", err)
				}
				restart = true
			case <-session.failure:
				restart = true
			case <-ticker.C:
				if session.pathChanged() {
					restart = true
				}
			}
		}
		ticker.Stop()
		session.close(false)
	}
}

func processFromPeer(connection *net.UnixConn) *notifier.Process {
	_, pid, ok := peerCredentials(connection)
	if !ok {
		return nil
	}
	process := notifier.NewProcess(pid)
	if process != nil {
		process.Source = "peercred"
	}
	return process
}

func proxyUnixSocket(reader net.Conn, writer net.Conn, process *notifier.Process, requestGPGCheck chan *notifier.Process) {
	defer (func() {
		reader.Close()
		writer.Close()
	})()

	buf := make([]byte, 10240)
	for {
		nr, err := reader.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		if _, err = writer.Write(data); err != nil {
			return
		}

		select {
		case requestGPGCheck <- process:
		default:
		}
	}
}
