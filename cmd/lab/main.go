// SPDX-License-Identifier: Apache-2.0
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const owner = "substrate-ax-firsthand"

type state struct {
	Root           string `json:"root"`
	Container      string `json:"container"`
	Image          string `json:"image"`
	Port           int    `json:"port"`
	ClusterCreated bool   `json:"clusterCreated"`
}
type inspection struct {
	Config struct {
		Labels map[string]string
		Image  string
	}
	State struct {
		Running   bool
		StartedAt string
	}
}
type launcher struct {
	state
	log          *os.File
	refreshTools bool
	axHome       string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	rootFlag := flag.String("root", ".", "companion repository directory")
	refreshTools := flag.Bool("refresh-toolbox", false, "replace only the owned toolbox after an image change; preserves the cluster and mounted files")
	port := flag.Int("port", 0, "local router port for setup (default 18080)")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		return errors.New("usage: go run -buildvcs=false ./cmd/lab setup|shell|test|smoke|cleanup|exec")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return errors.New("this edition supports Linux/amd64 with a local Docker daemon")
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	if err := validateRoot(root); err != nil {
		return err
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("install Docker and start its local daemon first")
	}
	if err := os.MkdirAll(filepath.Join(root, ".local"), 0755); err != nil {
		return err
	}
	boxID := sha256.Sum256([]byte(root))
	boxName := "firsthand-tools-" + hex.EncodeToString(boxID[:6])
	l := &launcher{state: state{Root: root, Container: boxName, Port: 18080}, refreshTools: *refreshTools}
	data, err := os.ReadFile(l.statePath())
	if err == nil {
		if err := json.Unmarshal(data, &l.state); err != nil {
			return fmt.Errorf("invalid launcher state: %w", err)
		}
		if l.Root != root || l.Container != boxName {
			return errors.New("launcher state belongs to another directory; refusing to retarget")
		}
	} else if !os.IsNotExist(err) {
		return err
	} else if args[0] != "setup" {
		return errors.New("run go run -buildvcs=false ./cmd/lab setup first")
	}
	if args[0] == "setup" {
		if *port != 0 {
			l.Port = *port
		}
		if l.Port < 1024 || l.Port > 65535 {
			return errors.New("router port must be between 1024 and 65535")
		}
		return l.setup()
	}
	if err := l.ensureContainer(); err != nil {
		return err
	}
	var command []string
	switch args[0] {
	case "shell":
		command = append([]string{"bash", "scripts/lab-shell.sh"}, args[1:]...)
	case "exec":
		command = args[1:]
		if len(command) == 0 {
			return errors.New("exec needs a command")
		}
	case "test":
		command = []string{"env", "CGO_ENABLED=1", "make", "test"}
	case "smoke":
		command = []string{"bash", "scripts/smoke.sh"}
	case "cleanup":
		command = []string{"env", "CONFIRM_TUTORIAL_CLEANUP=" + os.Getenv("CONFIRM_TUTORIAL_CLEANUP"), "bash", "scripts/cleanup.sh"}
	default:
		return fmt.Errorf("unknown lab command %q", args[0])
	}
	cmd := exec.Command("docker", l.execArgs(command, terminal())...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func validateRoot(root string) error {
	if strings.ContainsAny(root, ":,\n\r") {
		return errors.New("repository path cannot contain colons, commas, or newlines")
	}
	for _, path := range []string{"versions.env", "go.mod", "images/toolbox.Dockerfile"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			return fmt.Errorf("run from the companion repository root: %w", err)
		}
	}
	return nil
}
func terminal() bool {
	in, a := os.Stdin.Stat()
	out, b := os.Stdout.Stat()
	return a == nil && b == nil && in.Mode()&os.ModeCharDevice != 0 && out.Mode()&os.ModeCharDevice != 0
}
func (l *launcher) statePath() string { return filepath.Join(l.Root, ".local", "launcher.json") }
func (l *launcher) save() error {
	data, err := json.MarshalIndent(l.state, "", "  ")
	if err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(l.statePath()), ".launcher-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	_, writeErr := out.Write(append(data, '\n'))
	closeErr := out.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(out.Name(), l.statePath())
}
func dockerOutput(args ...string) ([]byte, error) { return exec.Command("docker", args...).Output() }
func checkOwner(labels map[string]string, root string) error {
	if labels["firsthand.owner"] != owner || labels["firsthand.root"] != root {
		return errors.New("toolbox ownership mismatch; refusing to adopt or replace it")
	}
	return nil
}
func (l *launcher) ensureContainer() error {
	data, err := dockerOutput("inspect", l.Container)
	if err != nil {
		return fmt.Errorf("toolbox unavailable; rerun setup to diagnose or recreate it: %w", err)
	}
	var boxes []inspection
	if err := json.Unmarshal(data, &boxes); err != nil || len(boxes) != 1 {
		return errors.New("cannot identify toolbox container")
	}
	if err := checkOwner(boxes[0].Config.Labels, l.Root); err != nil {
		return err
	}
	if boxes[0].Config.Image != l.Image {
		return errors.New("toolbox image differs from this setup; preserve it and use a fresh companion directory")
	}
	if !boxes[0].State.Running {
		if _, err := dockerOutput("start", l.Container); err != nil {
			return err
		}
		data, err := dockerOutput("inspect", l.Container)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &boxes); err != nil || len(boxes) != 1 || !boxes[0].State.Running {
			return errors.New("owned toolbox did not start")
		}
	}
	if boxes[0].State.StartedAt == "" {
		return errors.New("toolbox start epoch is unavailable")
	}
	l.axHome = tunnelHome(l.Root, boxes[0].State.StartedAt)
	return nil
}
func tunnelHome(root, epoch string) string {
	hash := sha256.Sum256([]byte(epoch))
	return filepath.Join(root, ".local", "ax", hex.EncodeToString(hash[:8]))
}
func (l *launcher) execArgs(command []string, tty bool) []string {
	args := []string{"exec", "-i"}
	if tty {
		args = append(args, "-t")
	}
	args = append(args, "--workdir", l.Root, "--env", "TUTORIAL_IN_TOOLBOX=1")
	if l.Port != 0 {
		args = append(args, "--env", fmt.Sprintf("TUTORIAL_PORT=%d", l.Port))
	}
	if l.axHome != "" {
		args = append(args, "--env", "TUTORIAL_AX_HOME="+l.axHome)
	}
	args = append(args, l.Container)
	return append(args, command...)
}
func (l *launcher) stage(name string, command ...string) error {
	start := time.Now()
	fmt.Println(name)
	fmt.Fprintf(l.log, "\n%s\n", name)
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = l.Root, l.log, l.log
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("%s failed (%w). Details: %s. Diagnose, then rerun setup; it never recreates an owned cluster", name, err, l.log.Name())
			}
			fmt.Printf("  ready (%s)\n", time.Since(start).Round(time.Second))
			return nil
		case <-ticker.C:
			fmt.Printf("  still working (%s); details in .local/setup.log\n", time.Since(start).Round(time.Second))
		}
	}
}
func (l *launcher) inBox(name string, command ...string) error {
	return l.stage(name, append([]string{"docker"}, l.execArgs(command, false)...)...)
}

func (l *launcher) setup() error {
	started := time.Now()
	if _, err := dockerOutput("info"); err != nil {
		return errors.New("Docker daemon is not reachable; check docker info and your permissions")
	}
	if !l.ClusterCreated {
		if _, err := os.Stat(filepath.Join(l.Root, ".local", "state.env")); err == nil {
			return errors.New("an earlier prepared lab exists here; preserve it and use a fresh companion directory for setup")
		}
	}
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" {
		data, err := dockerOutput("context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
		if err != nil {
			return err
		}
		endpoint = strings.TrimSpace(string(data))
	}
	if !strings.HasPrefix(endpoint, "unix://") {
		return errors.New("this edition requires a local, rootful Docker Unix socket, not remote Docker")
	}
	socket := strings.TrimPrefix(endpoint, "unix://")
	info, err := os.Stat(socket)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode()&os.ModeSocket == 0 {
		return errors.New("Docker endpoint is not a Unix socket")
	}
	if strings.Contains(socket, "/run/user/") {
		return errors.New("rootless Docker is not validated in this edition")
	}
	l.log, err = os.OpenFile(filepath.Join(l.Root, ".local", "setup.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer l.log.Close()
	if err := l.installBinary(); err != nil {
		return err
	}
	dockerfile, err := os.ReadFile(filepath.Join(l.Root, "images", "toolbox.Dockerfile"))
	if err != nil {
		return err
	}
	imageHash := sha256.Sum256(append(dockerfile, []byte(fmt.Sprintf("/%d/%d", os.Getuid(), os.Getgid()))...))
	l.Image = "firsthand-toolbox:" + hex.EncodeToString(imageHash[:6])
	if err := l.stage("[1/6] Provision the private development toolbox", "docker", "build", "--build-arg", fmt.Sprintf("LAB_UID=%d", os.Getuid()), "--build-arg", fmt.Sprintf("LAB_GID=%d", os.Getgid()), "-t", l.Image, "-f", "images/toolbox.Dockerfile", "."); err != nil {
		return err
	}
	if data, err := dockerOutput("inspect", l.Container); err == nil {
		var existing []inspection
		if err := json.Unmarshal(data, &existing); err != nil || len(existing) != 1 {
			return errors.New("cannot identify existing toolbox")
		}
		if err := checkOwner(existing[0].Config.Labels, l.Root); err != nil {
			return err
		}
		if existing[0].Config.Image != l.Image {
			if !l.refreshTools {
				return errors.New("toolbox image changed; review the update, then use --refresh-toolbox setup to replace only our owned tools container")
			}
			fmt.Println("  replacing only the owned toolbox; its forwards end, mounted files and Kind cluster are preserved")
			if _, err := dockerOutput("rm", "-f", l.Container); err != nil {
				return err
			}
		}
	}
	if _, err := dockerOutput("inspect", l.Container); err != nil {
		args := []string{"run", "-d", "--init", "--network=host", "--name", l.Container,
			"--label", "firsthand.owner=" + owner, "--label", "firsthand.root=" + l.Root,
			"--group-add", fmt.Sprint(stat.Gid), "--mount", "type=bind,src=" + socket + ",dst=/var/run/docker.sock",
			"--mount", "type=bind,src=" + l.Root + ",dst=" + l.Root, "--workdir", l.Root,
			"--env", "TUTORIAL_CONTEXT=kind-firsthand", "--env", "TUTORIAL_IN_TOOLBOX=1",
			"--env", "KUBECONFIG=" + filepath.Join(l.Root, ".local", "kubeconfig"),
			"--env", "GOCACHE=" + filepath.Join(l.Root, ".local", "cache", "go-build"),
			"--env", "GOMODCACHE=" + filepath.Join(l.Root, ".local", "cache", "go-mod"),
			"--env", "DOCKER_CONFIG=" + filepath.Join(l.Root, ".local", "docker"), l.Image}
		if _, err := dockerOutput(args...); err != nil {
			return err
		}
	}
	if err := l.ensureContainer(); err != nil {
		return err
	}
	if err := l.save(); err != nil {
		return err
	}
	if err := l.inBox("[2/6] Fetch the pinned Substrate and AX source", "bash", "scripts/sources.sh"); err != nil {
		return err
	}
	if !l.ClusterCreated {
		if err := l.inBox("[3/6] Create Kind and connect the local image registry", "bash", "scripts/bootstrap.sh", "--cluster-only"); err != nil {
			return err
		}
		l.ClusterCreated = true
		if err := l.save(); err != nil {
			return err
		}
	} else {
		if err := l.inBox("[3/6] Verify the owned Kind cluster (no recreation)", "kubectl", "--context", "kind-firsthand", "get", "nodes"); err != nil {
			return err
		}
	}
	if err := l.inBox("[4/6] Install Substrate, snapshot storage, and gVisor workers", "bash", "scripts/platform.sh"); err != nil {
		return err
	}
	if err := l.inBox("[5/6] Build the pinned CLIs, AX server, runner, and counter images", "bash", "scripts/prepare.sh"); err != nil {
		return err
	}
	if err := l.inBox("[6/6] Install AX and tutorial namespaces; wait for readiness", "bash", "scripts/install.sh"); err != nil {
		return err
	}
	fmt.Printf("Lab ready in %s. Enter ./bin/lab shell, then follow the article to create the counter's WorkerPool and ActorTemplate.\n", time.Since(started).Round(time.Second))
	fmt.Println("Private kubeconfig: .local/kubeconfig. Other kubeconfig contexts were not changed.")
	return nil
}

func (l *launcher) installBinary() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	dest := filepath.Join(l.Root, "bin", "lab")
	if path == dest {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.CreateTemp(filepath.Dir(dest), ".lab-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	_, copyErr := io.Copy(out, in)
	modeErr := out.Chmod(0755)
	closeErr := out.Close()
	if err := errors.Join(copyErr, modeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(out.Name(), dest)
}
