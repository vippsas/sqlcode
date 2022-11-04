--sqlcode:include-if two

create procedure [code].Foo
as begin
    select * from [code].AddTwoNumbers(1, 2)
end


