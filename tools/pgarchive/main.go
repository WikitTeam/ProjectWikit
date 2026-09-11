package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/macho"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"

	"github.com/WikitTeam/ProjectWikit/internal/pgbundle"
)

const (
	defaultDir = "internal/pgbundle/archive"
	mavenBase  = "https://repo1.maven.org/maven2/io/zonky/test/postgres"
)

type target struct {
	platform string
	source   string
	cpu      macho.Cpu
}

var targets = map[string]target{
	"windows/amd64": {platform: "windows-amd64", source: "d059b085ea761279769613678af306129b08a190fd16c4f6ef71a124c931af40"},
	"linux/amd64":   {platform: "linux-amd64", source: "63999c2914366d62c68e51794e597588c336e75ed2958d53c558e8deca5e13fc"},
	"linux/arm64":   {platform: "linux-arm64v8", source: "2eb24d67326da41a81064d116b0bce412dff543c835d6fcc1c099e4abcf18836"},
	"darwin/amd64":  {platform: "darwin-amd64", source: "f4db5df0b13e3e72149c0547ccd1eee177cfe67accb8c296e8d0e46ef3601f01", cpu: macho.CpuAmd64},
	"darwin/arm64":  {platform: "darwin-arm64v8", source: "f4db5df0b13e3e72149c0547ccd1eee177cfe67accb8c296e8d0e46ef3601f01", cpu: macho.CpuArm64},
}

var droppedDirs = []string{"include/", "share/doc/", "share/locale/", "lib/pgxs/", "lib/pkgconfig/"}

var droppedSuffixes = []string{".a", ".lib", ".pdb"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pgarchive: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("pgarchive", flag.ContinueOnError)
	goos := fs.String("goos", runtime.GOOS, "operating system the archive is for")
	goarch := fs.String("goarch", runtime.GOARCH, "architecture the archive is for")
	dir := fs.String("dir", defaultDir, "directory the archive is written into")
	from := fs.String("from", "", "jar already downloaded, instead of fetching it")
	force := fs.Bool("force", false, "rebuild the archive even when it is up to date")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	t, ok := targets[*goos+"/"+*goarch]
	if !ok {
		return fmt.Errorf("no PostgreSQL build for %s/%s", *goos, *goarch)
	}
	base := filepath.Join(*dir, "postgresql-"+*goos+"-"+*goarch)
	stamp := t.source + " " + pgbundle.Version + "\n"
	if !*force {
		if have, err := os.ReadFile(base + ".source"); err == nil && string(have) == stamp {
			if _, err := os.Stat(base + ".tar.zst"); err == nil {
				fmt.Println("up to date:", base+".tar.zst")
				return nil
			}
		}
	}

	jar, err := readJar(*from, t.platform)
	if err != nil {
		return err
	}
	txz, err := archiveInJar(jar)
	if err != nil {
		return err
	}
	if got := sum(txz); got != t.source {
		return fmt.Errorf("the PostgreSQL %s archive for %s has sha256 %s, want %s", pgbundle.Version, t.platform, got, t.source)
	}

	var out bytes.Buffer
	stats, err := repack(&out, txz, t.cpu)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(base+".tar.zst", out.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(base+".sha256", []byte(sum(out.Bytes())), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(base+".source", []byte(stamp), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d files, %d linked, %d dropped, %d MB unpacked, %d MB packed\n",
		base+".tar.zst", stats.files, stats.linked, stats.dropped, stats.unpacked>>20, out.Len()>>20)
	return nil
}

func readJar(from, platform string) ([]byte, error) {
	if from != "" {
		return os.ReadFile(from)
	}
	name := "embedded-postgres-binaries-" + platform
	url := fmt.Sprintf("%s/%s/%s/%s-%s.jar", mavenBase, name, pgbundle.Version, name, pgbundle.Version)
	fmt.Println("fetching", url)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func archiveInJar(jar []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(jar), int64(len(jar)))
	if err != nil {
		return nil, fmt.Errorf("open jar: %w", err)
	}
	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, ".txz") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, errors.New("the jar holds no .txz archive")
}

type stats struct {
	files, linked, dropped int
	unpacked               int64
}

func repack(w io.Writer, txz []byte, cpu macho.Cpu) (stats, error) {
	var st stats
	xr, err := xz.NewReader(bytes.NewReader(txz))
	if err != nil {
		return st, err
	}
	tr := tar.NewReader(xr)

	zw, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBestCompression), zstd.WithEncoderConcurrency(1))
	if err != nil {
		return st, err
	}
	tw := tar.NewWriter(zw)

	seen := map[string]string{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return st, err
		}
		name := strings.TrimPrefix(path.Clean(strings.TrimPrefix(h.Name, "./")), "/")
		if name == "." || dropped(name, h.Typeflag) {
			st.dropped++
			continue
		}

		out := &tar.Header{Name: name, ModTime: time.Unix(0, 0).UTC(), Mode: 0o644}
		switch h.Typeflag {
		case tar.TypeDir:
			out.Typeflag, out.Name, out.Mode = tar.TypeDir, name+"/", 0o755
			if err := tw.WriteHeader(out); err != nil {
				return st, err
			}
		case tar.TypeSymlink:
			out.Typeflag, out.Linkname = tar.TypeSymlink, h.Linkname
			if err := tw.WriteHeader(out); err != nil {
				return st, err
			}
		case tar.TypeReg:
			data, err := io.ReadAll(tr)
			if err != nil {
				return st, err
			}
			if cpu != 0 {
				data = thin(data, cpu)
			}
			if h.Mode&0o111 != 0 {
				out.Mode = 0o755
			}
			key := sum(data) + fmt.Sprint(out.Mode)
			if first, ok := seen[key]; ok {
				out.Typeflag, out.Linkname = tar.TypeLink, first
				if err := tw.WriteHeader(out); err != nil {
					return st, err
				}
				st.linked++
				continue
			}
			seen[key] = name
			out.Typeflag, out.Size = tar.TypeReg, int64(len(data))
			if err := tw.WriteHeader(out); err != nil {
				return st, err
			}
			if _, err := tw.Write(data); err != nil {
				return st, err
			}
			st.files++
			st.unpacked += int64(len(data))
		default:
			st.dropped++
		}
	}
	if err := tw.Close(); err != nil {
		return st, err
	}
	return st, zw.Close()
}

func dropped(name string, kind byte) bool {
	probe := name
	if kind == tar.TypeDir {
		probe += "/"
	}
	for _, d := range droppedDirs {
		if strings.HasPrefix(probe, d) {
			return true
		}
	}
	for _, s := range droppedSuffixes {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return strings.HasPrefix(name, "bin/wx")
}

func thin(data []byte, cpu macho.Cpu) []byte {
	ff, err := macho.NewFatFile(bytes.NewReader(data))
	if err != nil {
		return data
	}
	defer ff.Close()
	for _, arch := range ff.Arches {
		if arch.Cpu == cpu {
			return data[arch.Offset : arch.Offset+arch.Size]
		}
	}
	return data
}

func sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
