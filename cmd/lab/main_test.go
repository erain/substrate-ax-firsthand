// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOwnershipFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		labels map[string]string
		valid  bool
	}{
		{nil, false},
		{map[string]string{"firsthand.owner": owner, "firsthand.root": "/other"}, false},
		{map[string]string{"firsthand.owner": "someone-else", "firsthand.root": "/lab"}, false},
		{map[string]string{"firsthand.owner": owner, "firsthand.root": "/lab"}, true},
	} {
		if got := checkOwner(tc.labels, "/lab") == nil; got != tc.valid {
			t.Fatalf("labels %v: valid=%v", tc.labels, got)
		}
	}
}

func TestExecArgumentsDoNotInvokeHostShell(t *testing.T) {
	l := &launcher{state: state{Root: "/lab with spaces", Container: "our-box"}}
	command := []string{"bash", "scripts/entrypoint.sh", "ax", "ssh", "task-one", "--", "sh", "-ec", "echo '$HOME $(id)'"}
	want := append([]string{"exec", "-i", "--workdir", l.Root, "--env", "TUTORIAL_IN_TOOLBOX=1", l.Container}, command...)
	if got := l.execArgs(command, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	if got := l.execArgs(command, true); !reflect.DeepEqual(got[:3], []string{"exec", "-i", "-t"}) {
		t.Fatal(got)
	}
}

func TestTunnelCacheIsPrivateToToolboxStart(t *testing.T) {
	a := tunnelHome("/lab", "start-one")
	if a != tunnelHome("/lab", "start-one") {
		t.Fatal("cache path is not stable")
	}
	if a == tunnelHome("/lab", "start-two") || a == tunnelHome("/other", "start-one") {
		t.Fatal("cache collides across toolbox restarts or labs")
	}
	l := &launcher{state: state{Root: "/lab", Container: "owned"}, axHome: a}
	args := l.execArgs([]string{"true"}, false)
	if !reflect.DeepEqual(args[6:8], []string{"--env", "TUTORIAL_AX_HOME=" + a}) {
		t.Fatal(args)
	}
}

func TestRouterPortPassedToToolbox(t *testing.T) {
	l := &launcher{state: state{Root: "/lab", Container: "owned", Port: 28080}}
	args := l.execArgs([]string{"true"}, false)
	if !reflect.DeepEqual(args[6:8], []string{"--env", "TUTORIAL_PORT=28080"}) {
		t.Fatal(args)
	}
}

func TestStateRoundTripAndPermissions(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".local"), 0755); err != nil {
		t.Fatal(err)
	}
	l := &launcher{state: state{Root: root, Container: "owned", Image: "pinned", ClusterCreated: true}}
	if err := l.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(l.statePath())
	if err != nil {
		t.Fatal(err)
	}
	var got state
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != l.state {
		t.Fatal(got)
	}
	info, err := os.Stat(l.statePath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
}

func TestRepositoryValidation(t *testing.T) {
	root := t.TempDir()
	if validateRoot(root) == nil {
		t.Fatal("accepted empty directory")
	}
	if err := os.Mkdir(filepath.Join(root, "images"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", "versions.env", "images/toolbox.Dockerfile"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateRoot(root); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{root + ":other", root + ",other", root + "\nother"} {
		if validateRoot(bad) == nil {
			t.Fatalf("accepted unsafe mount path %q", bad)
		}
	}
}

func TestInstallBinary(t *testing.T) {
	l := &launcher{state: state{Root: t.TempDir()}}
	if err := l.installBinary(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(l.Root, "bin", "lab"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 || info.Size() == 0 {
		t.Fatal(info)
	}
	if err := l.installBinary(); err != nil {
		t.Fatal("atomic replacement failed:", err)
	}
}
