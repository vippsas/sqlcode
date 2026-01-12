package pgsql

// PostgreSQL reserved words (PostgreSQL 17)
//
// This list includes all reserved keywords from PostgreSQL 17.
// Reserved words cannot be used as identifiers without quoting.
//
// Source: https://www.postgresql.org/docs/17/sql-keywords-appendix.html
//
// Note: PostgreSQL distinguishes between:
//   - Reserved keywords (cannot be used as identifiers)
//   - Non-reserved keywords (can be used as identifiers in most contexts)
//
// This map contains only the reserved keywords (marked as "reserved" in the docs).
var reservedWords = map[string]struct{}{
	// A
	"all":        {},
	"analyse":    {},
	"analyze":    {},
	"and":        {},
	"any":        {},
	"array":      {},
	"as":         {},
	"asc":        {},
	"asymmetric": {},

	// B
	"binary": {},
	"both":   {},

	// C
	"case":              {},
	"cast":              {},
	"check":             {},
	"collate":           {},
	"collation":         {},
	"column":            {},
	"concurrently":      {},
	"constraint":        {},
	"create":            {},
	"cross":             {},
	"current_catalog":   {},
	"current_date":      {},
	"current_role":      {},
	"current_schema":    {},
	"current_time":      {},
	"current_timestamp": {},
	"current_user":      {},

	// D
	"default":        {},
	"deferrable":     {},
	"desc":           {},
	"distinct":       {},
	"do":             {},
	"using":          {},
	"else":           {},
	"end":            {},
	"except":         {},
	"false":          {},
	"fetch":          {},
	"for":            {},
	"foreign":        {},
	"from":           {},
	"full":           {},
	"grant":          {},
	"group":          {},
	"having":         {},
	"ilike":          {},
	"in":             {},
	"initially":      {},
	"inner":          {},
	"intersect":      {},
	"into":           {},
	"is":             {},
	"join":           {},
	"lateral":        {},
	"leading":        {},
	"left":           {},
	"like":           {},
	"limit":          {},
	"localtime":      {},
	"localtimestamp": {},
	"natural":        {},
	"not":            {},
	"null":           {},
	"offset":         {},
	"on":             {},
	"only":           {},
	"or":             {},
	"order":          {},
	"outer":          {},
	"over":           {},
	"overlaps":       {},
	"placing":        {},
	"primary":        {},
	"references":     {},
	"returning":      {},
	"right":          {},
	"select":         {},
	"session_user":   {},
	"similar":        {},
	"some":           {},
	"symmetric":      {},
	"table":          {},
	"then":           {},
	"to":             {},
	"trailing":       {},
	"true":           {},
	"union":          {},
	"unique":         {},
	"user":           {},
	"variadic":       {},
	"verbose":        {},
	"when":           {},
	"where":          {},
	"window":         {},
	"with":           {},
}
