-- This is from parser_test.go; but present here in order to test
-- how the CLI / library reports errors from the sqlcode parser

create function [code].One();
-- the following should give an error; not that One() depends on Two()...
-- (we don't parse body start/end yet)
create function [code].Two();
