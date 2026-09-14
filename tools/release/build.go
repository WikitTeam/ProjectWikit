package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const versionPackage = "github.com/WikitTeam/ProjectWikit/internal/version.Release"

var versionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

func buildCommand(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	version := fs.String("version", "", "release version, such as v1.0.0 or v0.0.0-dev.abc1234")
	out := fs.String("out", "dist", "directory the package is written into")
	skipFrontend := fs.Bool("skip-frontend", false, "use the page assets already in static/")
	skipFtml := fs.Bool("skip-ftml", false, "use the ftml library already in ftml-capi/target/release")
	skipPostgres := fs.Bool("skip-postgres", false, "use the PostgreSQL archive already in internal/pgbundle/archive")
	toolchain := fs.String("cargo-toolchain", defaultToolchain(), "rustup toolchain the ftml library is built with; empty uses the default")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if !versionPattern.MatchString(*version) {
		return fmt.Errorf("-version %q is not a version such as v1.0.0", *version)
	}
	if _, err := os.Stat(filepath.Join("cmd", "pwikit")); err != nil {
		return errors.New("run this from the repository root")
	}

	started := time.Now()
	if !*skipFrontend {
		if err := step("frontend", "frontend", nil, "yarn", "install", "--frozen-lockfile"); err != nil {
			return err
		}
		if err := step("frontend", "frontend", nil, "yarn", "build"); err != nil {
			return err
		}
	}
	if !*skipFtml {
		if err := buildFtml(*toolchain); err != nil {
			return err
		}
	}
	if err := step("ftml", ".", nil, "go", "run", "./tools/ftmllib", "-emit-metadata"); err != nil {
		return err
	}
	if !*skipPostgres {
		if err := step("postgres", ".", nil, "go", "run", "./tools/pgarchive"); err != nil {
			return err
		}
	}

	base := packageBase(*version, runtime.GOOS, runtime.GOARCH)
	stage := filepath.Join(*out, "stage", base)
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	binary := filepath.Join(stage, executableName(runtime.GOOS))
	env := []string{"CGO_ENABLED=1"}
	if err := step("pwikit", ".", env, "go", "build", "-tags", "bundle", "-trimpath",
		"-ldflags", "-s -w -X "+versionPackage+"="+*version,
		"-o", binary, "./cmd/pwikit"); err != nil {
		return err
	}
	if err := copyFile("LICENSE", filepath.Join(stage, "LICENSE"), 0o644); err != nil {
		return err
	}

	target := filepath.Join(*out, packageFile(*version, runtime.GOOS, runtime.GOARCH))
	if runtime.GOOS == "windows" {
		err := writeZip(target, filepath.Join(*out, "stage"), base)
		if err != nil {
			return err
		}
	} else if err := writeTarGz(target, filepath.Join(*out, "stage"), base); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(*out, "stage")); err != nil {
		return err
	}

	if err := copyFtmlLibrary(*out); err != nil {
		return err
	}

	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s, %d MB, in %s\n", target, info.Size()>>20, time.Since(started).Round(time.Second))
	return nil
}

func copyFtmlLibrary(out string) error {
	meta, err := os.ReadFile(filepath.Join("ftml-capi", "target", "release", "libftml_capi.json"))
	if err != nil {
		return err
	}
	var described struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(meta, &described); err != nil {
		return err
	}
	for _, name := range []string{"libftml_capi.a", "libftml_capi.json"} {
		ext := filepath.Ext(name)
		published := strings.TrimSuffix(name, ext) + "-" + described.Target + ext
		if err := copyFile(filepath.Join("ftml-capi", "target", "release", name), filepath.Join(out, published), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func defaultToolchain() string {
	if runtime.GOOS == "windows" {
		return "stable-x86_64-pc-windows-gnu"
	}
	return ""
}

func buildFtml(toolchain string) error {
	args := []string{}
	if toolchain != "" {
		args = append(args, "+"+toolchain)
	}
	args = append(args, "rustc", "--release", "--lib", "--", "--print", "native-static-libs")
	var env []string
	if runtime.GOOS == "windows" {
		env = append(env, "RUSTFLAGS="+strings.TrimSpace(os.Getenv("RUSTFLAGS")+" -C link-arg=-ladvapi32"))
	}
	return step("ftml", "ftml-capi", env, "cargo", args...)
}

func step(label, dir string, env []string, name string, args ...string) error {
	fmt.Printf("==> [%s] %s %s\n", label, name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s %s: %w", label, name, strings.Join(args, " "), err)
	}
	return nil
}

func copyFile(from, to string, mode os.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func packedMode(name string, info os.FileInfo) int64 {
	if info.IsDir() {
		return 0o755
	}
	if base := filepath.Base(name); base == "pwikit" || base == "pwikit.exe" {
		return 0o755
	}
	return 0o644
}

func writeTarGz(target, root, top string) error {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)

	walkErr := filepath.Walk(filepath.Join(root, top), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		header := &tar.Header{
			Name:    filepath.ToSlash(rel),
			Mode:    packedMode(path, info),
			ModTime: info.ModTime(),
			Format:  tar.FormatPAX,
		}
		if info.IsDir() {
			header.Typeflag = tar.TypeDir
			header.Name += "/"
			return tw.WriteHeader(header)
		}
		header.Typeflag = tar.TypeReg
		header.Size = info.Size()
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
	if walkErr != nil {
		return walkErr
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return file.Close()
}

func writeZip(target, root, top string) error {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	zw := zip.NewWriter(file)

	walkErr := filepath.Walk(filepath.Join(root, top), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
			_, err := zw.CreateHeader(header)
			return err
		}
		header.Method = zip.Deflate
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	if walkErr != nil {
		return walkErr
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return file.Close()
}
