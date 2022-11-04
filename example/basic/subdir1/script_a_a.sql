-- _a_a.sql, this should be ordered after _a but before a_a_a.sql
CREATE function [code].A_X_plus_B(@a int, @x int, @b int)
returns int as begin return [code].AddTwoNumbers(@a * @x, @b) end;

