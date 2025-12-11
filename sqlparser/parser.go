// Simple non-performant recursive descent parser for purposes of sqlcode; currently only
// supports the special @Enum declarations used by sqlcode. We only allow
// these on the top, and parsing will stop
// without any errors at the point hitting anything else.
package sqlparser

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var templateRoutineName string = "\ndeclare @RoutineName nvarchar(128)\nset @RoutineName = '%s'\n"

var supportedSqlExtensions []string = []string{".sql", ".pgsql"}

func CopyToken(s *Scanner, target *[]Unparsed) {
	*target = append(*target, CreateUnparsed(s))
}

// AdvanceAndCopy is like NextToken; advance to next token that is not whitespace and return
// Note: The 'go' and EOF tokens are *not* copied
func AdvanceAndCopy(s *Scanner, target *[]Unparsed) {
	for {
		tt := s.NextToken()
		switch tt {
		case EOFToken, BatchSeparatorToken:
			// do not copy
			return
		case WhitespaceToken, MultilineCommentToken, SinglelineCommentToken:
			// copy, and loop around
			CopyToken(s, target)
			continue
		default:
			// copy, and return
			CopyToken(s, target)
			return
		}
	}
}

func Parse(s *Scanner, result Document) {
	// Top-level parse; this focuses on splitting into "batches" separated
	// by 'go'.

	// CONVENTION:
	// All functions should expect `s` positioned on what they are documented
	// to consume/parse.
	//
	// Functions typically consume *after* the keyword that triggered their
	// invoication; e.g. parseCreate parses from first non-whitespace-token
	// *after* `create`.
	//
	// On return, `s` is positioned at the token that starts the next statement/
	// sub-expression. In particular trailing ';' and whitespace has been consumed.
	//
	// `s` will typically never be positioned on whitespace except in
	// whitespace-preserving parsing
	s.NextNonWhitespaceToken()
	err := result.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse document: %s: %e", s.file, err))
	}
	return
}

// ParseFileystems iterates through a list of filesystems and parses all supported
// SQL files and returns the combination of all of them.
//
// err will only return errors related to filesystems/reading. Errors
// related to parsing/sorting will be in result.Errors.
//
// ParseFilesystems will also sort create statements topologically.
func ParseFilesystems(fslst []fs.FS, includeTags []string) (filenames []string, result Document, err error) {
	// We are being passed several *filesystems* here. It may be easy to pass in the same
	// directory twice but that should not be encouraged, so if we get the same hash from
	// two files, return an error. Only files containing [code] in some way will be
	// considered here anyway

	hashes := make(map[[32]byte]string)

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
						return errors.New(fmt.Sprintf("file %s has exact same contents as %s (possibly in different filesystems)",
							pathDesc, existingPathDesc))
					}
					hashes[hash] = pathDesc

					fdoc := NewDocumentFromExtension(extension)
					Parse(&Scanner{input: string(buf), file: FileRef(path)}, fdoc)

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

// consider something a "sqlcode source file" if it contains [code]
// or a --sqlcode: header
var isSqlCodeRegex = regexp.MustCompile(`^--sqlcode:|\[code\]`)
