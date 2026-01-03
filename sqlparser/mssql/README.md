
Package mssql provides a T-SQL (Microsoft SQL Server) parser for the sqlcode library.

# Overview
This package implements a lexical scanner and document parser specifically designed
for T-SQL syntax. It is part of the sqlcode toolchain that manages SQL database
objects (procedures, functions, types) with dependency tracking and code generation.

# Architecture
The parser follows a two-layer architecture:
 1. Scanner (scanner.go): A lexical tokenizer that breaks T-SQL source into tokens.
    It handles T-SQL-specific constructs like N'unicode strings', [bracketed identifiers],
    and the GO batch separator.
 2. Document (document.go): A higher-level parser that processes token streams to
    extract CREATE statements, DECLARE constants, and dependency information.

# Token System
T-SQL tokens are divided into two categories:
  - Common tokens (defined in sqldocument): Shared across SQL dialects (e.g., parentheses,
    whitespace, identifiers). These use token type values 0-999.
  - T-SQL-specific tokens (defined in tokens.go): Dialect-specific tokens like
    VarcharLiteralToken ('...') and NVarcharLiteralToken (N'...'). These use values 1000-1999.

# Batch Separator Handling
T-SQL uses GO as a batch separator with special rules:
  - GO must appear at the start of a line (only whitespace/comments before it)
  - Nothing except whitespace may follow GO on the same line
  - GO is not a reserved word; it's a client tool command
The scanner tracks line position state to correctly identify GO as a BatchSeparatorToken
rather than an identifier. Malformed separators (GO followed by non-whitespace) are
reported as MalformedBatchSeparatorToken.

# Document Structure
The parser recognizes:
  - CREATE PROCEDURE/FUNCTION/TYPE statements in the [code] schema
  - DECLARE statements for constants (variables starting with @Enum, @Global, or @Const)
  - Dependencies between objects via [code].ObjectName references
  - Pragma comments (--sqlcode:...) for build-time directives

# Dependency Tracking
When parsing CREATE statements, the parser scans for [code].ObjectName patterns
to build a dependency graph. This enables topological sorting of objects so they
are created in the correct order during deployment.

# Error Recovery
The parser uses a recovery strategy that skips to the next statement-starting
keyword (CREATE, DECLARE, GO) when encountering syntax errors. This allows
partial parsing of files with errors while collecting all error messages.
