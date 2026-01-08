// Simple non-performant recursive descent parser for purposes of sqlcode; currently only
// supports the special @Enum declarations used by sqlcode. We only allow
// these on the top, and parsing will stop
// without any errors at the point hitting anything else.
package sqlparser

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/vippsas/sqlcode/sqlparser/mssql"
	"github.com/vippsas/sqlcode/sqlparser/sqldocument"
)

var (
	templateRoutineName    string   = "\ndeclare @RoutineName nvarchar(128)\nset @RoutineName = '%s'\n"
	supportedSqlExtensions []string = []string{".sql"}
	// consider something a "sqlcode source file" if it contains [code]
	// or a --sqlcode: header
	isSqlCodeRegex = regexp.MustCompile(`^--sqlcode:|\[code\]`)
)

// Based on the input file extension, create the appropriate Document type
func NewDocumentFromExtension(extension string) sqldocument.Document {
	switch extension {
	case ".sql":
		return &mssql.TSqlDocument{}
	default:
		panic("unhandled document type: " + extension)
	}
}

// ParseFileystems iterates through a list of filesystems and parses all supported
// SQL files and returns the combination of all of them.
//
// err will only return errors related to filesystems/reading. Errors
// related to parsing/sorting will be in result.Errors.
//
// ParseFilesystems will also sort create statements topologically.
func ParseFilesystems(fslst []fs.FS, includeTags []string) (filenames []string, result sqldocument.Document, err error) {
	// We are being passed several *filesystems* here. It may be easy to pass in the same
	// directory twice but that should not be encouraged, so if we get the same hash from
	// two files, return an error. Only files containing [code] in some way will be
	// considered here anyway

	hashes := make(map[[32]byte]string)

	if result == nil {
		result = &mssql.TSqlDocument{}
	}

	for fidx, fsys := range fslst {
		// WalkDir is in lexical order according to docs, so output should be stable
		err = fs.WalkDir(fsys, ".",
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// Skip over any hidden directories; in particular .git
				if strings.HasPrefix(path, ".") || strings.Contains(path, "/.") {
					return nil
				}

				extension := filepath.Ext(path)
				if !slices.Contains(supportedSqlExtensions, extension) {
					return nil
				}

				buf, err := fs.ReadFile(fsys, path)
				if err != nil {
					return err
				}

				// Sniff whether the file is a SQLCode file or not. We can NOT use the parser
				// for this, because the parser can be thrown off by errors, and we can't have
				// a system where files are suddenly ignored when there are syntax errors!
				// So using a more stable regex
				if isSqlCodeRegex.Find(buf) != nil {

					// protect against same file being referenced from 2 identical file systems..or just same file included twice
					pathDesc := fmt.Sprintf("fs[%d]:%s", fidx, path)
					hash := sha256.Sum256(buf)
					existingPathDesc, hashExists := hashes[hash]
					if hashExists {
						return fmt.Errorf("file %s has exact same contents as %s (possibly in different filesystems)",
							pathDesc, existingPathDesc)
					}
					hashes[hash] = pathDesc

					fdoc := NewDocumentFromExtension(extension)
					err = fdoc.Parse(buf, sqldocument.FileRef(path))
					if err != nil {
						return fmt.Errorf("error parsing file %s: %w", pathDesc, err)
					}

					// only include if include tags match
					if matchesIncludeTags(fdoc.PragmaIncludeIf(), includeTags) {
						filenames = append(filenames, pathDesc)
						result.Include(fdoc)
					}
				}
				return nil
			})
		if err != nil {
			return
		}
	}

	result.Sort()

	return
}

func matchesIncludeTags(required []string, got []string) bool {
	for _, r := range required {
		found := false
		for _, g := range got {
			if g == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func IsSqlcodeConstVariable(varname string) bool {
	return strings.HasPrefix(varname, "@Enum") ||
		strings.HasPrefix(varname, "@ENUM_") ||
		strings.HasPrefix(varname, "@enum_") ||
		strings.HasPrefix(varname, "@Const") ||
		strings.HasPrefix(varname, "@CONST_") ||
		strings.HasPrefix(varname, "@const_")
}
