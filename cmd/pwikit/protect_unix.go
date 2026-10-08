//go:build unix

package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/pgbundle"
	"github.com/WikitTeam/ProjectWikit/internal/service"
	"github.com/WikitTeam/ProjectWikit/internal/update"
)

var instanceEntries = []string{
	"LICENSE", "NOTICE", "README", "README.md",
	"pwikit", "pwikit.old", "pwikit.new", "pwikit.toml", "pwikit.toml.part",
	"files", "archive", "secrets", "pgdata", "postgres", "postgres.old", "postgres.lock",
	"locales", "backups", "logs", "update",
}

var accountDirs = []string{"update", "logs", "files", "archive", "backups"}

func protectInstance(p *paths.Paths, exe string) string {
	if os.Geteuid() != 0 {
		return ""
	}
	root := p.Root()
	top, ok := statOf(root)
	if !ok {
		return ""
	}
	running, _ := runningExecutable()
	ownCopy := update.IsPrivate(running)
	inside := filepath.Dir(exe) == root
	if top.Uid != 0 {
		if extra := strangers(root); len(extra) > 0 {
			if ownCopy {
				return fmt.Sprintf("%s also holds %s, which pwikit did not put there, so pwikit left its owner as it is and another account can still change what root reads and writes there",
					root, strings.Join(extra, ", "))
			}
			return fmt.Sprintf("%s also holds %s, which pwikit did not put there, so pwikit left its owner as it is and another account can still replace %s",
				root, strings.Join(extra, ", "), exe)
		}
		uid, gid := int(top.Uid), int(top.Gid)
		for _, name := range accountDirs {
			dir := filepath.Join(root, name)
			if _, err := os.Lstat(dir); err == nil {
				continue
			}
			if err := os.Mkdir(dir, 0o755); err != nil {
				return fmt.Sprintf("create %s: %v", dir, err)
			}
			if err := os.Lchown(dir, uid, gid); err != nil {
				return fmt.Sprintf("hand %s to its account: %v", dir, err)
			}
		}
	}
	targets := []string{p.Config()}
	if inside {
		targets = append(targets, exe)
	}
	if top.Uid != 0 {
		targets = append(targets, root)
	}
	for _, path := range targets {
		if err := rootHolds(path); err != nil {
			return fmt.Sprintf("hand %s to root: %v", path, err)
		}
	}
	if ownCopy {
		return ""
	}
	return exposure(exe)
}

func rootHolds(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a link", path)
	}
	if err := os.Lchown(path, 0, -1); err != nil {
		return err
	}
	return os.Chmod(path, info.Mode().Perm()&^0o022)
}

func exposure(exe string) string {
	for path := exe; ; path = filepath.Dir(path) {
		st, ok := statOf(path)
		if !ok {
			return ""
		}
		info, err := os.Stat(path)
		if err != nil {
			return ""
		}
		if st.Uid != 0 || (info.Mode().Perm()&0o022 != 0 && info.Mode()&os.ModeSticky == 0) {
			return fmt.Sprintf("root runs %s, and another account can replace it through %s", exe, path)
		}
		if path == filepath.Dir(path) {
			return ""
		}
	}
}

func strangers(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || slices.Contains(instanceEntries, name) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	if len(out) > 3 {
		out = append(out[:3], "…")
	}
	return out
}

func statOf(path string) (*syscall.Stat_t, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return st, ok
}

func releaseInstance(root, exe string, uid, gid int) {
	top, ok := statOf(root)
	if !ok || top.Uid != 0 {
		return
	}
	if acct, _, ok := update.Account(root); !ok || int(acct) != uid {
		return
	}
	for _, path := range []string{root, filepath.Join(root, "pwikit.toml"), exe} {
		if path == exe && filepath.Dir(exe) != root {
			continue
		}
		os.Lchown(path, uid, gid)
	}
}

func settingsFileOpenTo(path string) string {
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Stat(p)
		if err != nil {
			return p + " cannot be checked"
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return p + " cannot be checked"
		}
		if st.Uid != 0 {
			return p + " belongs to " + accountName(st.Uid)
		}
		if info.Mode().Perm()&0o022 != 0 && info.Mode()&os.ModeSticky == 0 {
			return p + " can be changed by other accounts"
		}
		if p == filepath.Dir(p) {
			return ""
		}
	}
}

func accountName(uid uint32) string {
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
		return u.Username
	}
	return "uid " + strconv.FormatUint(uint64(uid), 10)
}

func prepareInstall(spec *service.Spec) error {
	if os.Geteuid() != 0 {
		return nil
	}
	if update.NoExec(spec.Root) {
		return fmt.Errorf("%s is on a filesystem mounted noexec, so the programs pwikit unpacks there cannot run. "+
			"Mount it without noexec, or move the pwikit directory", spec.Root)
	}
	if spec.User == pgbundle.SystemAccountName {
		if err := dedicate(spec); err != nil {
			return err
		}
	} else if u, err := service.ServiceUser(spec.User); err == nil {
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		if uid != 0 {
			releaseInstance(spec.Root, spec.Executable, uid, gid)
		}
	}
	rootService := false
	if u, err := service.ServiceUser(spec.User); err == nil && u.Uid == "0" {
		rootService = true
	}
	if len(spec.UpdateArgs) == 0 && !rootService {
		return nil
	}
	private, err := update.PrivateDir(spec.Root)
	if err != nil || private == "" {
		return err
	}
	bin := filepath.Join(private, update.PrivateBin, update.ExecutableName(runtime.GOOS))
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		return err
	}
	if spec.Executable != bin {
		if err := update.CopyFile(spec.Executable, bin); err != nil {
			return fmt.Errorf("keep root's copy of pwikit: %w", err)
		}
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		return err
	}
	if err := update.RecordInstance(private, spec.Root, spec.Executable); err != nil {
		return fmt.Errorf("record where pwikit is installed: %w", err)
	}
	if len(spec.UpdateArgs) > 0 {
		spec.UpdateExecutable = bin
	}
	if rootService {
		spec.RunExecutable = bin
	}
	return nil
}

func forgetInstance(root string) {
	if os.Geteuid() != 0 || root == "" {
		return
	}
	private := update.PrivatePath(root)
	if update.IsPrivate(private) {
		os.RemoveAll(private)
	}
}

func pinnedMirror(p *paths.Paths, mirror string) bool {
	rec, ok := update.ReadInstance(update.PrivatePath(p.Root()))
	if !ok {
		return false
	}
	dir, err := os.OpenRoot(p.Root())
	if err != nil {
		return false
	}
	defer dir.Close()
	top, err := dir.Stat(".")
	if err != nil || !rootOwned(top) {
		return false
	}
	if st, ok := top.Sys().(*syscall.Stat_t); !ok || uint64(st.Dev) != rec.Dev || uint64(st.Ino) != rec.Ino {
		return false
	}
	f, err := dir.Open(filepath.Base(p.Config()))
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !rootOwned(info) {
		return false
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	cfg, err := config.Parse(data, p.Config())
	return err == nil && strings.TrimSuffix(cfg.Update.Mirror, "/") == strings.TrimSuffix(mirror, "/")
}

func rootOwned(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == 0 && info.Mode().Perm()&0o022 == 0
}

func configLinkTrusted(path string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Uid == 0 {
		return nil
	}
	return fmt.Errorf("%s is a link another account made, so pwikit will not read it as root", path)
}

func dedicate(spec *service.Spec) error {
	u, err := pgbundle.SystemAccount(spec.Root)
	if err != nil {
		return err
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	root := spec.Root
	if err := pgbundle.RememberRole(filepath.Join(root, "pgdata")); err != nil {
		return err
	}
	for _, name := range []string{"files", "archive", "backups", "logs", "update", "secrets", "pgdata"} {
		dir := filepath.Join(root, name)
		if _, err := os.Lstat(dir); err == nil {
			continue
		}
		mode := os.FileMode(0o755)
		if name == "secrets" || name == "pgdata" {
			mode = 0o700
		}
		if err := os.Mkdir(dir, mode); err != nil {
			return err
		}
	}
	lock, err := os.OpenFile(filepath.Join(root, "postgres.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	lock.Close()
	if _, err := pgbundle.Unpack(filepath.Join(root, "postgres")); err != nil {
		return err
	}
	cfg := filepath.Join(root, "pwikit.toml")
	if _, err := config.WriteTemplate(cfg); err != nil {
		return err
	}
	state := []string{filepath.Join(root, "pgdata")}
	for _, path := range spec.State {
		if path != cfg {
			state = append(state, path)
		}
	}
	spec.State = state
	for _, path := range []string{filepath.Join(root, "postgres"), filepath.Join(root, "postgres.lock")} {
		filepath.WalkDir(path, func(p string, _ fs.DirEntry, err error) error {
			if err == nil {
				os.Lchown(p, uid, gid)
			}
			return nil
		})
	}
	targets := []string{root, cfg}
	if filepath.Dir(spec.Executable) == root {
		targets = append(targets, spec.Executable)
	}
	for _, path := range targets {
		if err := rootHolds(path); err != nil {
			return fmt.Errorf("hand %s to root: %w", path, err)
		}
	}
	if err := os.Lchown(cfg, 0, gid); err != nil {
		return err
	}
	return os.Chmod(cfg, 0o640)
}

func unpackPostgres(args []string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	p, err := paths.New(dataDirArg(args))
	if err != nil {
		return err
	}
	if _, err := pgbundle.Unpack(p.Postgres()); err != nil {
		return err
	}
	return update.ChownTree(p.Postgres(), p.PGData())
}

func prepareStart(root string) {
	if os.Geteuid() != 0 || root == "" {
		return
	}
	top, ok := statOf(root)
	if !ok || top.Uid != 0 {
		return
	}
	bin := filepath.Join(update.PrivatePath(root), update.PrivateBin, update.ExecutableName(runtime.GOOS))
	if _, err := os.Stat(bin); err != nil {
		return
	}
	cmd := exec.Command(bin, "unpack-postgres", "-data-dir", root)
	cmd.Dir = root
	cmd.Run()
}
