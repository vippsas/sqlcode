--sqlcode:include-if one

-- This should be ordered first (filename is never used)

create FUNcTIoN [code].AddTwoNumbers(@x int, @y int) returns int
as begin
    return @x + @y
end

go


-- foobar

create

    FUNcTIoN
    [code].AddTwoNumbersV2(@x int, @y int) returns

        int as begin return @x + @y end;

go


create type [code].MyType as table (x int not null primary key);
