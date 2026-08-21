package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	libraryName  = "libftml_capi.a"
	metadataName = "libftml_capi.json"
	defaultDir   = "ftml-capi/target/release"
)

type metadata struct {
	ABI    int    `json:"abi"`
	FTML   string `json:"ftml"`
	Target string `json:"target"`
	SHA256 string `json:"sha256"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ftmllib: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("ftmllib", flag.ContinueOnError)
	dir := fs.String("dir", defaultDir, "directory holding the static library")
	baseURL := fs.String("url", os.Getenv("PWIKIT_FTML_LIB_URL"), "base URL to download a missing library from")
	target := fs.String("target", defaultTarget(), "target triple the library must be built for")
	emit := fs.Bool("emit-metadata", false, "write metadata describing the library already in -dir")
	abi := fs.Int("abi", 0, "ABI version the build expects")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *abi == 0 {
		v, err := readABIVersion()
		if err != nil {
			return err
		}
		*abi = v
	}

	if *emit {
		return emitMetadata(*dir, *abi, *target)
	}
	return ensure(*dir, *baseURL, *target, *abi)
}

func defaultTarget() string {
	if runtime.GOOS == "windows" {
		return "x86_64-pc-windows-gnu"
	}
	return runtime.GOARCH + "-" + runtime.GOOS
}

func readABIVersion() (int, error) {
	data, err := os.ReadFile(filepath.Join("ftml-capi", "ABI_VERSION"))
	if err != nil {
		return 0, fmt.Errorf("read ABI_VERSION: %w", err)
	}
	version, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse ABI_VERSION: %w", err)
	}
	return version, nil
}

func ensure(dir, baseURL, target string, abi int) error {
	libPath := filepath.Join(dir, libraryName)

	if _, err := os.Stat(libPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if baseURL == "" {
			return fmt.Errorf("%s is missing and no -url was given; build it with\n"+
				"    cd ftml-capi && cargo build --release", libPath)
		}
		if err := download(dir, baseURL, target); err != nil {
			return err
		}
	}

	meta, err := readMetadata(dir)
	if err != nil {
		return err
	}
	return verify(libPath, meta, target, abi)
}

func verify(libPath string, meta metadata, target string, abi int) error {
	if meta.ABI != abi {
		return fmt.Errorf("%s provides ABI %d, this source tree expects %d; fetch a matching library or rebuild ftml-capi",
			libPath, meta.ABI, abi)
	}
	if meta.Target != target {
		return fmt.Errorf("%s was built for %s, not %s", libPath, meta.Target, target)
	}
	sum, err := hashFile(libPath)
	if err != nil {
		return err
	}
	if sum != meta.SHA256 {
		return fmt.Errorf("%s has sha256 %s, metadata records %s", libPath, sum, meta.SHA256)
	}
	fmt.Printf("%s ok: abi %d, ftml %s, %s\n", libPath, meta.ABI, meta.FTML, meta.Target)
	return nil
}

func readMetadata(dir string) (metadata, error) {
	var meta metadata
	path := filepath.Join(dir, metadataName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return meta, fmt.Errorf("%s is missing; run this tool with -emit-metadata after building the library", path)
		}
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, fmt.Errorf("parse %s: %w", path, err)
	}
	return meta, nil
}

func emitMetadata(dir string, abi int, target string) error {
	libPath := filepath.Join(dir, libraryName)
	sum, err := hashFile(libPath)
	if err != nil {
		return err
	}
	meta := metadata{ABI: abi, FTML: ftmlVersion(), Target: target, SHA256: sum}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, metadataName)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", path)
	return nil
}

func ftmlVersion() string {
	data, err := os.ReadFile(filepath.Join("ftml", "Cargo.toml"))
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "version") {
			_, value, ok := strings.Cut(line, "=")
			if !ok {
				break
			}
			return strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return "unknown"
}

func download(dir, baseURL, target string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	for _, name := range []string{libraryName, metadataName} {
		url := strings.TrimSuffix(baseURL, "/") + "/" + target + "/" + name
		if err := fetch(client, url, filepath.Join(dir, name)); err != nil {
			return err
		}
		fmt.Printf("downloaded %s\n", url)
	}
	return nil
}

func fetch(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	tmp := dest + ".part"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
