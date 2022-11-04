-- line below is not at top

--sqlcode:include-if three

-- Ordered after script_a_a.sql

-- This is a comment with [code] in it

  /*comment*/CREATE PROCEDURE [code].SelectAX_Plus_B(@a int, @x int, @b int)
as begin
    declare @myvar varchar(max) = 'using [code] in a string should not be replaced'
    select [code].A_X_plus_B(@a, @x, @b)
end
