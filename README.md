# sqlcode -- manage stored procedures/functions

If you have a substantial amount of SQL in your service, should
you write that code as a SQL strings compiled into the binary, or as a set
of stored procedures/functions? Both approaches have some big drawbacks;
this tool tries to bring the benefits  of both.

1) Write stored procedures, functions and types *only* (permanent tables will be prevented)
   in `*.sql`-files in your code base

2) Call `sqlcode up mydb:mybranch` to upload it to a temporary schema `[code@mybranch]`
   for interactive debugging on a database.

3) When your service starts it ensures that the code is uploaded to a
   schema dedicated for the given version of your code (`[code@ba432abf]`).
  
Benefits over strings in backend source code:

* Clean error messages with filename and line number
* Fast debugging, verification and profiling of your queries against live databases
  before deploying your service

Benefits over stored procedures in migration files:

* Smooth rollout of upgrades and rollbacks with multiple concurrent versions
  running at the same time, one for each service. With standard migrations achieving
  this is at least a manual process with manually versioned procedures.

* Git diffs works properly

## INSTALL

To install the CLI tool do something along these lines:
```sh
$ go build -o ~/.local/bin/sqlcode ./cli
```

To fetch the Go library:
```sh
$ go get github.com/simukka/sqlcode
```

## HOWTO

### Step 1

Put your SQL code in `*.sql`-files in your repo. The directory
you place it in is irrelevant, `sqlcode` will simply scan the entire subtree.

For now, `sqlcode` assumes a single global namespace, i.e. for each database
and code repo there is a single namespace for stored procedures and
functions. If you want namespacing then for now that must
be done as part of function/procedure/type names.

The SQL file has a declaration header comment and should otherwise
contain creation of enums, types, procedures or functions in a virtual
schema `[code]`; always written with the brackets.
**Do not create tables/indexes**. Like this:

```sql
-- Global constants
declare
    @EnumGasoline int = 1,
    @EnumDiesel int = 2,
    @EnumElectric int = 3;

-- Batch markers supported
go

create type [code].MyType as table (x int not null primary key);
go
-- All stored procedures/functions/types should go in the [code] schema;
-- ALWAYS INCLUDE THE BRACKETS.
create function [code].Add(@a int, @b int) returns int
as begin 
    return @a + @b + @EnumDiesel;
end;
```

**Step 2**

Verify that the `sqlcode` preprocessor does what you think it should do:

```shell
$ sqlcode build mypackage debug123
create type [code].MyType as table (x int not null primary key);
go
create function [code@debug123].Add(@a int, @b int) returns int
as begin
return @a + @b + 2/*=@EnumDiesel*/;
end;

go 

create function [code@debug123].Report(@t [code].MyType readonly) returns table
as return select sum(x) as Sum from @t;

go
```

What SQLCode does is:

1) Replace uses of global constant declarations with their literal values
2) Replace `[code]` with `[code@<deployment suffix>]` everywhere (but not
   in strings or comments)
3) Concatenate all `create procedure/function/type` statements *in the right order* so that 
   the SQL files can refer to names declared elsewhere without any issues. 

This command is mainly useful for debugging `sqlcode` itself and make sure
you understand how it works; `sqlcode build` is not normally
used in your workflow.

### Step 3
Install the `sqlcode` SQL library by running the migrations
in [migrations](migrations) in your ordinary SQL migration pipeline.
This installs some stored procedures that are used by the utilities
below; in the `sqlcode` schema. Please read through the
migration files for more information before installing.

Currently, only Microsoft SQL is supported.

You also have to do something like this for your service users and also
any humans that should manually upload using `sqlcode up`:
```sql
alter role [sqlcode-deploy-role] add member your_service_user;
```

Services or users that only execute code and do not upload it can instead
be added like to `[sqlcode-execute-role]`.

NOTE: The above roles control full access to all `[code@...]` schemas jointly;
something to keep in mind if multiple services use the same DB.


### Step 4

Create a file `sqlcode.yaml` in the root of your repo listing the
databases to access. Assuming AAD token login:

```sql
databases:
    test:
        connection: sqlserver://mytestenv.database.windows.net:1433?database=myservice
    prod:
        connection: sqlserver://myprodenv.database.windows.net:1433?database=myservice
```

Then upload the code manually for some quick tests:
```bash
$ sqlcode up test:mybranch
```

Here `test` refers to the database, and `mybranch` is a throwaway name you
choose to not interfer with other users of the database, and other work
you are doing yourself.

Then you can fire up DataGrip / SMSS / ... and check query plans,
manually including the schema suffix:
```sql
-- debug session:
select [code@mybranch].Add(1, 2)
```


### Step 5 (Go-specific)

Your backend service adds a call to automatically upload code
during startup (currently only Go is supported). It will upload it to a
schema with a name determined by hash of the SQL files involved.

First we embed the SQL source file in the compiled binary using the `//go:embed`
feature, and call `sqlcode.MustInclude` to load and parse the SQL files
at program startup:
```go
package mypackage

import (
    "embed"
    "github.com/simukka/sqlcode"
)

//go:embed *.sql
//go:embed subdir/*.sql
var SQLFS embed.FS
var SQL = sqlcode.MustInclude(SQLFS)

var EnumMyGlobal = SQL.MustIntConst("@EnumMyGlobal")
```

Then you have to make sure the code is uploaded when you have a DB connection
pool ready; this can be during program startup, but it can also be 
The function will return quickly without network traffic if it has already
been done for this DB by this process:
Upload at the start of your program:
```go
func main() {
	// ...
	err := SQL.EnsureUploaded(ctx, dbc)
	// ...
}
```
This is pretty much equivalent to
```shell
$ sqlcode up prod:dc8f9910de0d
```
that is, the schema suffix is chosen as service name provided,
current date, and a hash of the SQL. However, `SQL.EnsureUploaded`
will not upload a second time if it has already been done,
while `sqlcode up` will drop the target schema and re-upload (replace).

### Step 6

Once code has been uploaded, you invoke the same pre-processors on whatever
SQL strings you have left inline in your code in order to support
calling into `[code]`:

```go
var addSql = SQL.Patch(`select [code].Add(1, 2)`)
func myfunc() {
	// ... 
	_, err := dbc.ExecContext(ctx, addSql) 
	// ...
}
```

### Step 7

As days pass more and more SQL code is uploaded and some cleanup
is needed. These features are not added to this library yet though.


## Feature guide

## Security model

The security conscious user should make sure to review
`migratins/0001.sqlcode.sql` and fully understand what is going on.

Note that the `sqlcode.CreateCodeSchema` and `sqlcode.DropCodeSchema` stored
procedures are signed and operate as `db_owner`; any user who gets granted
`execute` on these procedures may create and drop `[code@...]` schemas at will.
Any injection attack bugs in these procedures could provide points for
privilege escalation, although no such bugs are known.

The following security measures are implemented:

* The `[code@...]` schemas are owned by a special user `[sqlcode-user-with-no-permissions]`
  which are granted no rights. This disables one avenue where stored procedures
  gets the same permissions as the owner of the stored procedure, and one is
  left with the permissions of the caller of the procedure.

* During upload of the SQL code, the user `[sqlcode-deploy-sandbox-user]` is
  impersonated to reduce privileges for the operation. This is both for security,
  and also so that the user does not have `create table`, `create index`
  permissions in the database.

## Enum/global constant support

If a `*.sql`-file contains code like the following at the top level

```sql
declare @EnumFoo int = 3
```
...then this will be treated as a global constant, and inlined everywhere
as `3/* =@EnumFoo */`. We also support `varchar(max)`, even correcting
error messages for the shifted line numbers when literals contain newlines.

Such global constants must start with either of
`@Enum`, `@ENUM_`, `@enum_`, `@Const`, `@CONST_`, `@const_`.
(This is experimental; in the future perhaps we will instead use `@$` or similar
for SQLCode global constants).

Global constants must be declared in a batch of their own.
If a source file *only* contains such global constants, you have to
have at least one pragma in it, such as this,
```sql
--sqlcode:

declare @EnumFoo int = 3
```
so that the file itself will be picked up by SQLCode; SQLCode picks up files
that either contains `[code]`, or starts with `--sqlcode`.

The CLI command `sqlcode constants` will dump all the `declare @EnumFoo ..`
statements in the subtree for easy copy+paste of everything into your
debugging session.

## Introspection and annotations

It can be convenient to annotate stored procedures/functions with some metadata
available to a backend. For instance, consider a backend that automatically
exposes endpoints based on SQL queries. To aid building such things the
Go parser, when used as a library, makes introspection data available
such as the name of the function, arguments etc. (features here are added
as needed).

A comment immediately before a create statement is treated like a "docstring"
and is available in the node representing the create statement in the DOM.
There is also a convention and DOM support for an embedded YAML document
in the docstring; the lines containing the YAML document should be prefixed
by `!-- ` (note the space): Example:

```sql
-- Returns JSON to return for the /myentity GET endpoint
--
--! timeoutMs: 400
--! runAs: myserviceuser
--! this:
--!  - is: ["a", "yaml", "document"]
create procedure [code].[GET:/myentity] (@entityID bigint) as begin 
    select 
        [name] = Name
    from myschema.MyEntity
    for json path
end
```

