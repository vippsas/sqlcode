package pgsql

// PostgreSQL keywords (PostgreSQL 17)
//
// Generated from pg_get_keywords() output.
//
// Source: https://www.postgresql.org/docs/17/sql-keywords-appendix.html
//
// Keyword categories:
//   - R: Reserved (cannot be used as identifiers without quoting)
//   - T: Reserved (can be function or type name)
//   - C: Unreserved (cannot be function or type name)
//   - U: Unreserved (can be bare label)

// KeywordCategory represents the category of a PostgreSQL keyword.
type KeywordCategory string

const (
	// Reserved keywords that cannot be used as identifiers
	CategoryReserved KeywordCategory = "R"
	// Reserved keywords that can be function or type names
	CategoryTypeFunc KeywordCategory = "T"
	// Unreserved keywords that cannot be function or type names
	CategoryColName KeywordCategory = "C"
	// Unreserved keywords
	CategoryUnreserved KeywordCategory = "U"
)

// KeywordInfo contains metadata about a PostgreSQL keyword.
type KeywordInfo struct {
	Category     KeywordCategory
	CanBareLabel bool
}

// AllKeywords contains all PostgreSQL 17 keywords with their metadata.
var AllKeywords = map[string]KeywordInfo{
	// A
	"abort":         {CategoryUnreserved, true},
	"absent":        {CategoryUnreserved, true},
	"absolute":      {CategoryUnreserved, true},
	"access":        {CategoryUnreserved, true},
	"action":        {CategoryUnreserved, true},
	"add":           {CategoryUnreserved, true},
	"admin":         {CategoryUnreserved, true},
	"after":         {CategoryUnreserved, true},
	"aggregate":     {CategoryUnreserved, true},
	"all":           {CategoryReserved, true},
	"also":          {CategoryUnreserved, true},
	"alter":         {CategoryUnreserved, true},
	"always":        {CategoryUnreserved, true},
	"analyse":       {CategoryReserved, true},
	"analyze":       {CategoryReserved, true},
	"and":           {CategoryReserved, true},
	"any":           {CategoryReserved, true},
	"array":         {CategoryReserved, false},
	"as":            {CategoryReserved, false},
	"asc":           {CategoryReserved, true},
	"asensitive":    {CategoryUnreserved, true},
	"assertion":     {CategoryUnreserved, true},
	"assignment":    {CategoryUnreserved, true},
	"asymmetric":    {CategoryReserved, true},
	"at":            {CategoryUnreserved, true},
	"atomic":        {CategoryUnreserved, true},
	"attach":        {CategoryUnreserved, true},
	"attribute":     {CategoryUnreserved, true},
	"authorization": {CategoryTypeFunc, true},

	// B
	"backward": {CategoryUnreserved, true},
	"before":   {CategoryUnreserved, true},
	"begin":    {CategoryUnreserved, true},
	"between":  {CategoryColName, true},
	"bigint":   {CategoryColName, true},
	"binary":   {CategoryTypeFunc, true},
	"bit":      {CategoryColName, true},
	"boolean":  {CategoryColName, true},
	"both":     {CategoryReserved, true},
	"breadth":  {CategoryUnreserved, true},
	"by":       {CategoryUnreserved, true},

	// C
	"cache":             {CategoryUnreserved, true},
	"call":              {CategoryUnreserved, true},
	"called":            {CategoryUnreserved, true},
	"cascade":           {CategoryUnreserved, true},
	"cascaded":          {CategoryUnreserved, true},
	"case":              {CategoryReserved, true},
	"cast":              {CategoryReserved, true},
	"catalog":           {CategoryUnreserved, true},
	"chain":             {CategoryUnreserved, true},
	"char":              {CategoryColName, false},
	"character":         {CategoryColName, false},
	"characteristics":   {CategoryUnreserved, true},
	"check":             {CategoryReserved, true},
	"checkpoint":        {CategoryUnreserved, true},
	"class":             {CategoryUnreserved, true},
	"close":             {CategoryUnreserved, true},
	"cluster":           {CategoryUnreserved, true},
	"coalesce":          {CategoryColName, true},
	"collate":           {CategoryReserved, true},
	"collation":         {CategoryTypeFunc, true},
	"column":            {CategoryReserved, true},
	"columns":           {CategoryUnreserved, true},
	"comment":           {CategoryUnreserved, true},
	"comments":          {CategoryUnreserved, true},
	"commit":            {CategoryUnreserved, true},
	"committed":         {CategoryUnreserved, true},
	"compression":       {CategoryUnreserved, true},
	"concurrently":      {CategoryTypeFunc, true},
	"conditional":       {CategoryUnreserved, true},
	"configuration":     {CategoryUnreserved, true},
	"conflict":          {CategoryUnreserved, true},
	"connection":        {CategoryUnreserved, true},
	"constraint":        {CategoryReserved, true},
	"constraints":       {CategoryUnreserved, true},
	"content":           {CategoryUnreserved, true},
	"continue":          {CategoryUnreserved, true},
	"conversion":        {CategoryUnreserved, true},
	"copy":              {CategoryUnreserved, true},
	"cost":              {CategoryUnreserved, true},
	"create":            {CategoryReserved, false},
	"cross":             {CategoryTypeFunc, true},
	"csv":               {CategoryUnreserved, true},
	"cube":              {CategoryUnreserved, true},
	"current":           {CategoryUnreserved, true},
	"current_catalog":   {CategoryReserved, true},
	"current_date":      {CategoryReserved, true},
	"current_role":      {CategoryReserved, true},
	"current_schema":    {CategoryTypeFunc, true},
	"current_time":      {CategoryReserved, true},
	"current_timestamp": {CategoryReserved, true},
	"current_user":      {CategoryReserved, true},
	"cursor":            {CategoryUnreserved, true},
	"cycle":             {CategoryUnreserved, true},
	// D
	"data": {CategoryUnreserved, true}, "database": {CategoryUnreserved, true}, "day": {CategoryUnreserved, false}, "deallocate": {CategoryUnreserved, true}, "dec": {CategoryColName, true}, "decimal": {CategoryColName, true}, "declare": {CategoryUnreserved, true}, "default": {CategoryReserved, true}, "defaults": {CategoryUnreserved, true}, "deferrable": {CategoryReserved, true}, "deferred": {CategoryUnreserved, true}, "definer": {CategoryUnreserved, true}, "delete": {CategoryUnreserved, true}, "delimiter": {CategoryUnreserved, true}, "delimiters": {CategoryUnreserved, true}, "depends": {CategoryUnreserved, true}, "depth": {CategoryUnreserved, true}, "desc": {CategoryReserved, true}, "detach": {CategoryUnreserved, true}, "dictionary": {CategoryUnreserved, true}, "disable": {CategoryUnreserved, true}, "discard": {CategoryUnreserved, true}, "distinct": {CategoryReserved, true}, "do": {CategoryReserved, true}, "document": {CategoryUnreserved, true}, "domain": {CategoryUnreserved, true}, "double": {CategoryUnreserved, true}, "drop": {CategoryUnreserved, true},
	// E
	"each": {CategoryUnreserved, true}, "else": {CategoryReserved, true}, "empty": {CategoryUnreserved, true}, "enable": {CategoryUnreserved, true}, "encoding": {CategoryUnreserved, true}, "encrypted": {CategoryUnreserved, true}, "end": {CategoryReserved, true}, "enforced": {CategoryUnreserved, true}, "enum": {CategoryUnreserved, true}, "error": {CategoryUnreserved, true}, "escape": {CategoryUnreserved, true}, "event": {CategoryUnreserved, true}, "except": {CategoryReserved, false}, "exclude": {CategoryUnreserved, true}, "excluding": {CategoryUnreserved, true}, "exclusive": {CategoryUnreserved, true}, "execute": {CategoryUnreserved, true}, "exists": {CategoryColName, true}, "explain": {CategoryUnreserved, true}, "expression": {CategoryUnreserved, true}, "extension": {CategoryUnreserved, true}, "external": {CategoryUnreserved, true}, "extract": {CategoryColName, true},
	// F
	"false": {CategoryReserved, true}, "family": {CategoryUnreserved, true}, "fetch": {CategoryReserved, false}, "filter": {CategoryUnreserved, false}, "finalize": {CategoryUnreserved, true}, "first": {CategoryUnreserved, true}, "float": {CategoryColName, true}, "following": {CategoryUnreserved, true}, "for": {CategoryReserved, false}, "force": {CategoryUnreserved, true}, "foreign": {CategoryReserved, true}, "format": {CategoryUnreserved, true}, "forward": {CategoryUnreserved, true}, "freeze": {CategoryTypeFunc, true}, "from": {CategoryReserved, false}, "full": {CategoryTypeFunc, true}, "function": {CategoryUnreserved, true}, "functions": {CategoryUnreserved, true},
	// G
	"generated": {CategoryUnreserved, true}, "global": {CategoryUnreserved, true}, "grant": {CategoryReserved, false}, "granted": {CategoryUnreserved, true}, "greatest": {CategoryColName, true}, "group": {CategoryReserved, false}, "grouping": {CategoryColName, true}, "groups": {CategoryUnreserved, true},
	// H
	"handler": {CategoryUnreserved, true}, "having": {CategoryReserved, false}, "header": {CategoryUnreserved, true}, "hold": {CategoryUnreserved, true}, "hour": {CategoryUnreserved, false},
	// I
	"identity": {CategoryUnreserved, true}, "if": {CategoryUnreserved, true}, "ilike": {CategoryTypeFunc, true}, "immediate": {CategoryUnreserved, true}, "immutable": {CategoryUnreserved, true}, "implicit": {CategoryUnreserved, true}, "import": {CategoryUnreserved, true}, "in": {CategoryReserved, true}, "include": {CategoryUnreserved, true}, "including": {CategoryUnreserved, true}, "increment": {CategoryUnreserved, true}, "indent": {CategoryUnreserved, true}, "index": {CategoryUnreserved, true}, "indexes": {CategoryUnreserved, true}, "inherit": {CategoryUnreserved, true}, "inherits": {CategoryUnreserved, true}, "initially": {CategoryReserved, true}, "inline": {CategoryUnreserved, true}, "inner": {CategoryTypeFunc, true}, "inout": {CategoryColName, true}, "input": {CategoryUnreserved, true}, "insensitive": {CategoryUnreserved, true}, "insert": {CategoryUnreserved, true}, "instead": {CategoryUnreserved, true}, "int": {CategoryColName, true}, "integer": {CategoryColName, true}, "intersect": {CategoryReserved, false}, "interval": {CategoryColName, true}, "into": {CategoryReserved, false}, "invoker": {CategoryUnreserved, true}, "is": {CategoryTypeFunc, true}, "isnull": {CategoryTypeFunc, false}, "isolation": {CategoryUnreserved, true},
	// J
	"join": {CategoryTypeFunc, true}, "json": {CategoryColName, true}, "json_array": {CategoryColName, true}, "json_arrayagg": {CategoryColName, true}, "json_exists": {CategoryColName, true}, "json_object": {CategoryColName, true}, "json_objectagg": {CategoryColName, true}, "json_query": {CategoryColName, true}, "json_scalar": {CategoryColName, true}, "json_serialize": {CategoryColName, true}, "json_table": {CategoryColName, true}, "json_value": {CategoryColName, true},
	// K
	"keep": {CategoryUnreserved, true}, "key": {CategoryUnreserved, true}, "keys": {CategoryUnreserved, true},
	// L
	"label": {CategoryUnreserved, true}, "language": {CategoryUnreserved, true}, "large": {CategoryUnreserved, true}, "last": {CategoryUnreserved, true}, "lateral": {CategoryReserved, true}, "leading": {CategoryReserved, true}, "leakproof": {CategoryUnreserved, true}, "least": {CategoryColName, true}, "left": {CategoryTypeFunc, true}, "level": {CategoryUnreserved, true}, "like": {CategoryTypeFunc, true}, "limit": {CategoryReserved, false}, "listen": {CategoryUnreserved, true}, "load": {CategoryUnreserved, true}, "local": {CategoryUnreserved, true}, "localtime": {CategoryReserved, true}, "localtimestamp": {CategoryReserved, true}, "location": {CategoryUnreserved, true}, "lock": {CategoryUnreserved, true}, "locked": {CategoryUnreserved, true}, "logged": {CategoryUnreserved, true},
	// M
	"mapping": {CategoryUnreserved, true}, "match": {CategoryUnreserved, true}, "matched": {CategoryUnreserved, true}, "materialized": {CategoryUnreserved, true}, "maxvalue": {CategoryUnreserved, true}, "merge": {CategoryUnreserved, true}, "merge_action": {CategoryColName, true}, "method": {CategoryUnreserved, true}, "minute": {CategoryUnreserved, false}, "minvalue": {CategoryUnreserved, true}, "mode": {CategoryUnreserved, true}, "month": {CategoryUnreserved, false}, "move": {CategoryUnreserved, true},
	// N
	"name": {CategoryUnreserved, true}, "names": {CategoryUnreserved, true}, "national": {CategoryColName, true}, "natural": {CategoryTypeFunc, true}, "nchar": {CategoryColName, true}, "nested": {CategoryUnreserved, true}, "new": {CategoryUnreserved, true}, "next": {CategoryUnreserved, true}, "nfc": {CategoryUnreserved, true}, "nfd": {CategoryUnreserved, true}, "nfkc": {CategoryUnreserved, true}, "nfkd": {CategoryUnreserved, true}, "no": {CategoryUnreserved, true}, "none": {CategoryColName, true}, "normalize": {CategoryColName, true}, "normalized": {CategoryUnreserved, true}, "not": {CategoryReserved, true}, "nothing": {CategoryUnreserved, true}, "notify": {CategoryUnreserved, true}, "notnull": {CategoryTypeFunc, false}, "nowait": {CategoryUnreserved, true}, "null": {CategoryReserved, true}, "nullif": {CategoryColName, true}, "nulls": {CategoryUnreserved, true}, "numeric": {CategoryColName, true},
	// O
	"object": {CategoryUnreserved, true}, "objects": {CategoryUnreserved, true}, "of": {CategoryUnreserved, true}, "off": {CategoryUnreserved, true}, "offset": {CategoryReserved, false}, "oids": {CategoryUnreserved, true}, "old": {CategoryUnreserved, true}, "omit": {CategoryUnreserved, true}, "on": {CategoryReserved, false}, "only": {CategoryReserved, true}, "operator": {CategoryUnreserved, true}, "option": {CategoryUnreserved, true}, "options": {CategoryUnreserved, true}, "or": {CategoryReserved, true}, "order": {CategoryReserved, false}, "ordinality": {CategoryUnreserved, true}, "others": {CategoryUnreserved, true}, "out": {CategoryColName, true}, "outer": {CategoryTypeFunc, true}, "over": {CategoryUnreserved, false}, "overlaps": {CategoryTypeFunc, false}, "overlay": {CategoryColName, true}, "overriding": {CategoryUnreserved, true}, "owned": {CategoryUnreserved, true}, "owner": {CategoryUnreserved, true},
	// P
	"parallel": {CategoryUnreserved, true}, "parameter": {CategoryUnreserved, true}, "parser": {CategoryUnreserved, true}, "partial": {CategoryUnreserved, true}, "partition": {CategoryUnreserved, true}, "passing": {CategoryUnreserved, true}, "password": {CategoryUnreserved, true}, "path": {CategoryUnreserved, true}, "period": {CategoryUnreserved, true}, "placing": {CategoryReserved, true}, "plan": {CategoryUnreserved, true}, "plans": {CategoryUnreserved, true}, "policy": {CategoryUnreserved, true}, "position": {CategoryColName, true}, "preceding": {CategoryUnreserved, true}, "precision": {CategoryColName, false}, "prepare": {CategoryUnreserved, true}, "prepared": {CategoryUnreserved, true}, "preserve": {CategoryUnreserved, true}, "primary": {CategoryReserved, true}, "prior": {CategoryUnreserved, true}, "privileges": {CategoryUnreserved, true}, "procedural": {CategoryUnreserved, true}, "procedure": {CategoryUnreserved, true}, "procedures": {CategoryUnreserved, true}, "program": {CategoryUnreserved, true}, "publication": {CategoryUnreserved, true},
	// Q
	"quote": {CategoryUnreserved, true}, "quotes": {CategoryUnreserved, true},
	// R
	"range": {CategoryUnreserved, true}, "read": {CategoryUnreserved, true}, "real": {CategoryColName, true}, "reassign": {CategoryUnreserved, true}, "recursive": {CategoryUnreserved, true}, "ref": {CategoryUnreserved, true}, "references": {CategoryReserved, true}, "referencing": {CategoryUnreserved, true}, "refresh": {CategoryUnreserved, true}, "reindex": {CategoryUnreserved, true}, "relative": {CategoryUnreserved, true}, "release": {CategoryUnreserved, true}, "rename": {CategoryUnreserved, true}, "repeatable": {CategoryUnreserved, true}, "replace": {CategoryUnreserved, true}, "replica": {CategoryUnreserved, true}, "reset": {CategoryUnreserved, true}, "restart": {CategoryUnreserved, true}, "restrict": {CategoryUnreserved, true}, "return": {CategoryUnreserved, true}, "returning": {CategoryReserved, false}, "returns": {CategoryUnreserved, true}, "revoke": {CategoryUnreserved, true}, "right": {CategoryTypeFunc, true}, "role": {CategoryUnreserved, true}, "rollback": {CategoryUnreserved, true}, "rollup": {CategoryUnreserved, true}, "routine": {CategoryUnreserved, true}, "routines": {CategoryUnreserved, true}, "row": {CategoryColName, true}, "rows": {CategoryUnreserved, true}, "rule": {CategoryUnreserved, true},
	// S
	"savepoint": {CategoryUnreserved, true}, "scalar": {CategoryUnreserved, true}, "schema": {CategoryUnreserved, true}, "schemas": {CategoryUnreserved, true}, "scroll": {CategoryUnreserved, true}, "search": {CategoryUnreserved, true}, "second": {CategoryUnreserved, false}, "security": {CategoryUnreserved, true}, "select": {CategoryReserved, true}, "sequence": {CategoryUnreserved, true}, "sequences": {CategoryUnreserved, true}, "serializable": {CategoryUnreserved, true}, "server": {CategoryUnreserved, true}, "session": {CategoryUnreserved, true}, "session_user": {CategoryReserved, true}, "set": {CategoryUnreserved, true}, "setof": {CategoryColName, true}, "sets": {CategoryUnreserved, true}, "share": {CategoryUnreserved, true}, "show": {CategoryUnreserved, true}, "similar": {CategoryTypeFunc, true}, "simple": {CategoryUnreserved, true}, "skip": {CategoryUnreserved, true}, "smallint": {CategoryColName, true}, "snapshot": {CategoryUnreserved, true}, "some": {CategoryReserved, true}, "source": {CategoryUnreserved, true}, "sql": {CategoryUnreserved, true}, "stable": {CategoryUnreserved, true}, "standalone": {CategoryUnreserved, true}, "start": {CategoryUnreserved, true}, "statement": {CategoryUnreserved, true}, "statistics": {CategoryUnreserved, true}, "stdin": {CategoryUnreserved, true}, "stdout": {CategoryUnreserved, true}, "storage": {CategoryUnreserved, true}, "stored": {CategoryUnreserved, true}, "strict": {CategoryUnreserved, true}, "string": {CategoryUnreserved, true}, "strip": {CategoryUnreserved, true}, "subscription": {CategoryUnreserved, true},
	"substring":   {CategoryColName, true},
	"support":     {CategoryUnreserved, true},
	"symmetric":   {CategoryReserved, true},
	"sysid":       {CategoryUnreserved, true},
	"system":      {CategoryUnreserved, true},
	"system_user": {CategoryReserved, true},

	// T
	"table":       {CategoryReserved, true},
	"tables":      {CategoryUnreserved, true},
	"tablesample": {CategoryTypeFunc, true},
	"tablespace":  {CategoryUnreserved, true},
	"target":      {CategoryUnreserved, true},
	"temp":        {CategoryUnreserved, true},
	"template":    {CategoryUnreserved, true},
	"temporary":   {CategoryUnreserved, true},
	"text":        {CategoryUnreserved, true},
	"then":        {CategoryReserved, true},
	"ties":        {CategoryUnreserved, true},
	"time":        {CategoryColName, true},
	"timestamp":   {CategoryColName, true},
	"to":          {CategoryReserved, false},
	"trailing":    {CategoryReserved, true},
	"transaction": {CategoryUnreserved, true},
	"transform":   {CategoryUnreserved, true},
	"treat":       {CategoryColName, true},
	"trigger":     {CategoryUnreserved, true},
	"trim":        {CategoryColName, true},
	"true":        {CategoryReserved, true},
	"truncate":    {CategoryUnreserved, true},
	"trusted":     {CategoryUnreserved, true},
	"type":        {CategoryUnreserved, true},
	"types":       {CategoryUnreserved, true},

	// U
	"uescape":       {CategoryUnreserved, true},
	"unbounded":     {CategoryUnreserved, true},
	"uncommitted":   {CategoryUnreserved, true},
	"unconditional": {CategoryUnreserved, true},
	"unencrypted":   {CategoryUnreserved, true},
	"union":         {CategoryReserved, false},
	"unique":        {CategoryReserved, true},
	"unknown":       {CategoryUnreserved, true},
	"unlisten":      {CategoryUnreserved, true},
	"unlogged":      {CategoryUnreserved, true},
	"until":         {CategoryUnreserved, true},
	"update":        {CategoryUnreserved, true},
	"user":          {CategoryReserved, true},
	"using":         {CategoryReserved, true},

	// V
	"vacuum":    {CategoryUnreserved, true},
	"valid":     {CategoryUnreserved, true},
	"validate":  {CategoryUnreserved, true},
	"validator": {CategoryUnreserved, true},
	"value":     {CategoryUnreserved, true},
	"values":    {CategoryColName, true},
	"varchar":   {CategoryColName, true},
	"variadic":  {CategoryReserved, true},
	"varying":   {CategoryUnreserved, false},
	"verbose":   {CategoryTypeFunc, true},
	"version":   {CategoryUnreserved, true},
	"view":      {CategoryUnreserved, true},
	"views":     {CategoryUnreserved, true},
	"virtual":   {CategoryUnreserved, true},
	"volatile":  {CategoryUnreserved, true},

	// W
	"when":       {CategoryReserved, true},
	"where":      {CategoryReserved, false},
	"whitespace": {CategoryUnreserved, true},
	"window":     {CategoryReserved, false},
	"with":       {CategoryReserved, false},
	"within":     {CategoryUnreserved, false},
	"without":    {CategoryUnreserved, false},
	"work":       {CategoryUnreserved, true},
	"wrapper":    {CategoryUnreserved, true},
	"write":      {CategoryUnreserved, true},

	// X
	"xml":           {CategoryUnreserved, true},
	"xmlattributes": {CategoryColName, true},
	"xmlconcat":     {CategoryColName, true},
	"xmlelement":    {CategoryColName, true},
	"xmlexists":     {CategoryColName, true},
	"xmlforest":     {CategoryColName, true},
	"xmlnamespaces": {CategoryColName, true},
	"xmlparse":      {CategoryColName, true},
	"xmlpi":         {CategoryColName, true},
	"xmlroot":       {CategoryColName, true},
	"xmlserialize":  {CategoryColName, true},
	"xmltable":      {CategoryColName, true},

	// Y
	"year": {CategoryUnreserved, false},
	"yes":  {CategoryUnreserved, true},

	// Z
	"zone": {CategoryUnreserved, true},
}

// reservedWords contains only fully reserved keywords (category R and T)
// for use by the scanner to identify reserved words.
var reservedWords = func() map[string]struct{} {
	result := make(map[string]struct{})
	for word, info := range AllKeywords {
		if info.Category == CategoryReserved || info.Category == CategoryTypeFunc {
			result[word] = struct{}{}
		}
	}
	return result
}()

// IsReserved returns true if the given word is a reserved keyword.
func IsReserved(word string) bool {
	_, ok := reservedWords[word]
	return ok
}

// GetKeywordInfo returns the keyword information for the given word.
// Returns nil if the word is not a keyword.
func GetKeywordInfo(word string) *KeywordInfo {
	if info, ok := AllKeywords[word]; ok {
		return &info
	}
	return nil
}
