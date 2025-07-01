package mapfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type MapFS map[string]string

var _ fs.FS = (*MapFS)(nil)

func (m MapFS) Open(filename string) (fs.File, error) {
	if filename == "." {
		var entries []fs.DirEntry
		for base, fullpath := range m {
			info, err := os.Stat(fullpath)
			if err != nil {
				continue
			}
			entries = append(entries, fileDirEntry{name: base, info: info})
		}
		return &virtualDir{
			entries: entries,
			pos:     0,
		}, nil
	}

	if _, ok := m[filename]; !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, filename)
	}
	return os.Open(m[filename])
}

func (m MapFS) Add(path string) {
	filename := filepath.Base(path)
	m[filename] = path
}

// virtualDir implements fs.File + ReadDirFile
type virtualDir struct {
	entries []fs.DirEntry
	pos     int
}

func (d *virtualDir) Stat() (fs.FileInfo, error) {
	return dirInfo{name: ".", mode: fs.ModeDir}, nil
}

func (d *virtualDir) Read([]byte) (int, error) {
	return 0, io.EOF // directories have no data
}

func (d *virtualDir) Close() error {
	return nil
}

func (d *virtualDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.pos >= len(d.entries) {
		return nil, io.EOF
	}
	if n <= 0 || d.pos+n > len(d.entries) {
		n = len(d.entries) - d.pos
	}
	entries := d.entries[d.pos : d.pos+n]
	d.pos += n
	return entries, nil
}

// fileDirEntry implements fs.DirEntry
type fileDirEntry struct {
	name string
	info os.FileInfo
}

func (e fileDirEntry) Name() string               { return e.name }
func (e fileDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e fileDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e fileDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

// dirInfo is a simple FileInfo for the root dir
type dirInfo struct {
	name string
	mode fs.FileMode
}

func (d dirInfo) Name() string       { return d.name }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return d.mode }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return d.mode.IsDir() }
func (d dirInfo) Sys() interface{}   { return nil }
