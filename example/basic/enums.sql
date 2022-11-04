-- Comment

/*

 declare @EnumCool int = 4;
 */
declare    @EnumA varchar(max) = N'this is a

test',    @EnumB tinyint = 5,    @ENUM_C bigint = 435;

go

create procedure [code].DummyProc as
begin
    select @EnumA;
end
