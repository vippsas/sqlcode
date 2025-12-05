-- consts can be 

-- sqlcode 
-- we define schemas per deployment
-- uploading all stored functions/procedures/types/consts to a schema
-- pods are restarted/deployed per deployment

-- aaa bbb
   3   1

(iof increase in errors, stop deployment)
-- aaa bbb
   3   0
-- aaa bbb
   1   2
-- aaa bbb
   0   3
-- 

-- ++ both mssql and pgsql have the same architecture with schemas and stored functions/procedures

-- Q: constants?
-- we have the same constants defined in both mssql and pggsql

create procedure [code].test() -- expands to code@aaa.test
language plpgsql
as $$
begin
    perform 1;
end;
$$;