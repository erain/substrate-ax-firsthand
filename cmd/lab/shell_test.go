// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func shellFixture(t *testing.T, scripts ...string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "lab with spaces")
	for _, dir := range []string{"scripts", "bin", ".local/rendered"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range append([]string{"common.sh"}, scripts...) {
		data, err := os.ReadFile(filepath.Join("..", "..", "scripts", name))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, filepath.Join(root, "scripts", name), string(data))
	}
	writeFixture(t, filepath.Join(root, "versions.env"), "# Test fixture\n")
	writeFixture(t, filepath.Join(root, ".local", "state.env"), "TUTORIAL_CONTEXT=kind-firsthand\nTUTORIAL_PORT=28080\n")
	writeFixture(t, filepath.Join(root, ".local", "kubeconfig"), "# Private test config\n")
	return root
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0700); err != nil {
		t.Fatal(err)
	}
}

func runFixture(t *testing.T, root, script string, env []string, args ...string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", append([]string{filepath.Join(root, "scripts", script)}, args...)...)
	cmd.Env = append(os.Environ(), "TUTORIAL_IN_TOOLBOX=1", "PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"))
	cmd.Env = append(cmd.Env, env...)
	return cmd.CombinedOutput()
}

func TestLabShellEnvironmentAndArguments(t *testing.T) {
	root := shellFixture(t, "lab-shell.sh")
	axHome := filepath.Join(root, ".local", "ax", "start-one")
	literal := "spaces 'quotes' $(not-a-command)"
	output, err := runFixture(t, root, "lab-shell.sh", []string{
		"AX_SERVER=http://do-not-use",
		"KUBECONFIG=/do-not-use",
		"TUTORIAL_AX_HOME=" + axHome,
	}, "bash", "--noprofile", "--norc", "-c",
		`printf '%s\n' "$PATH" "$KUBECONFIG" "$AX_HOME" "$TUTORIAL_CONTEXT" "$TUTORIAL_PORT" "${AX_SERVER-unset}" "$1"`, "test", literal)
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) != 7 || !strings.HasPrefix(lines[0], filepath.Join(root, "bin")+":") {
		t.Fatalf("unexpected shell environment: %q", lines)
	}
	want := []string{filepath.Join(root, ".local", "kubeconfig"), axHome, "kind-firsthand", "28080", "unset", literal}
	for i, value := range want {
		if lines[i+1] != value {
			t.Fatalf("line %d: got %q, want %q", i+1, lines[i+1], value)
		}
	}
}

func TestLabShellRefusesMissingPrivateState(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  bool
		axHome  string
		message string
	}{
		{"kubeconfig", false, "private-cache", "private kubeconfig"},
		{"tunnel-cache", true, "", "private tunnel cache"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := shellFixture(t, "lab-shell.sh")
			if !tc.config {
				if err := os.Remove(filepath.Join(root, ".local", "kubeconfig")); err != nil {
					t.Fatal(err)
				}
			}
			output, err := runFixture(t, root, "lab-shell.sh", []string{"TUTORIAL_AX_HOME=" + tc.axHome}, "true")
			if err == nil || !strings.Contains(string(output), tc.message) {
				t.Fatalf("expected refusal %q, got %v: %s", tc.message, err, output)
			}
		})
	}
}

func TestPlatformInstallLeavesCounterCreationToReader(t *testing.T) {
	root := shellFixture(t, "install.sh")
	logPath := filepath.Join(root, ".local", "calls")
	writeFixture(t, filepath.Join(root, "bin", "kubectl"), `#!/usr/bin/env bash
printf 'kubectl %s\n' "$*" >> "$LAB_TEST_LOG"
case "$*" in *jsonpath=*) printf substrate-ax-tutorial ;; esac
`)
	writeFixture(t, filepath.Join(root, "bin", "kubectl-ate"), `#!/usr/bin/env bash
printf 'ate %s\n' "$*" >> "$LAB_TEST_LOG"
`)
	writeFixture(t, filepath.Join(root, "scripts", "ax"), `#!/usr/bin/env bash
printf 'ax %s\n' "$*" >> "$LAB_TEST_LOG"
`)
	output, err := runFixture(t, root, "install.sh", []string{"LAB_TEST_LOG=" + logPath})
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	calls := string(data)
	for _, required := range []string{"apply -f " + root + "/.local/rendered/lab-namespace.yaml", "apply -f " + root + "/.local/rendered/ax-control-plane.yaml", "ax get tasks -a firsthand"} {
		if !strings.Contains(calls, required) {
			t.Fatalf("missing %q in calls:\n%s", required, calls)
		}
	}
	for _, forbidden := range []string{"worker-pool.yaml", "actor-template", "deployment/firsthand-workers"} {
		if strings.Contains(calls, forbidden) {
			t.Fatalf("setup created counter resources (%q):\n%s", forbidden, calls)
		}
	}
}

func TestCounterFixturesRefuseUnownedNamespace(t *testing.T) {
	root := shellFixture(t, "counter-fixtures.sh")
	writeFixture(t, filepath.Join(root, "bin", "kubectl"), `#!/usr/bin/env bash
case "$*" in
  *"get namespace firsthand-lab"*) printf someone-else ;;
  *) echo "Unexpected write: $*" >&2; exit 99 ;;
esac
`)
	output, err := runFixture(t, root, "counter-fixtures.sh", nil)
	if err == nil || !strings.Contains(string(output), "Ownership mismatch for firsthand-lab") {
		t.Fatalf("expected ownership refusal, got %v: %s", err, output)
	}
}

func TestSmokeStopsItsRouterProcess(t *testing.T) {
	root := shellFixture(t)
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "smoke.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	start, end := strings.Index(script, "router_pid=\n"), strings.Index(script, "\nate create actor")
	if start < 0 || end <= start {
		t.Fatal("cannot find the smoke test's router lifecycle block")
	}
	writeFixture(t, filepath.Join(root, "scripts", "router-test.sh"),
		"#!/usr/bin/env bash\nsource \"$(dirname \"$0\")/common.sh\"\nload_state\n"+script[start:end]+"\n")
	pidPath := filepath.Join(root, ".local", "router-pid")
	writeFixture(t, filepath.Join(root, "bin", "kubectl"), `#!/usr/bin/env bash
printf '%s\n' "$BASHPID" > "$LAB_TEST_ROUTER_PID"
trap 'exit 0' TERM
while :; do sleep 0.1; done
`)
	writeFixture(t, filepath.Join(root, "bin", "curl"), `#!/usr/bin/env bash
test -s "$LAB_TEST_ROUTER_PID"
`)
	output, err := runFixture(t, root, "router-test.sh", []string{"LAB_TEST_ROUTER_PID=" + pidPath})
	pidData, readErr := os.ReadFile(pidPath)
	if readErr != nil {
		t.Fatalf("router never started: %v (script: %v, %s)", readErr, err, output)
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if parseErr != nil || pid <= 0 {
		t.Fatalf("invalid router PID: %q", pidData)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGTERM) })
	if err != nil {
		t.Fatalf("router lifecycle failed: %v: %s", err, output)
	}
	if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
		t.Fatalf("router process %d remains after smoke exit: %v", pid, err)
	}
}
