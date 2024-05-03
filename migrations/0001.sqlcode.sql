-- This user will own the sqlcode schemas, so that created stored procedures
-- by default is owned by a user without permissions; this means stored procedures
-- will not get more permissions than the user calling them already has
create user [sqlcode-user-with-no-permissions] without login;

-- This role will be granted execute permissions on all sqlcode schemas;
-- useful e.g. for humans logging in to debug
create role [sqlcode-execute-role];

-- Role for calling CreateCodeSchema / DropCodeSchema; the role will also be granted
-- control over all schemas created this way.
create role [sqlcode-deploy-role];


-- Make a user that *only* has this role. During deploys we drop permissions to this user so that we can
-- more safely deploy code with restricted permissions.
create user [sqlcode-deploy-sandbox-user] without login;
alter role [sqlcode-deploy-role] add member [sqlcode-deploy-sandbox-user];

-- If you are a member of [sqlcode-deploy-role], then you get to impersonate a user that *only* has that role;
-- seems fair...
grant impersonate on user::[sqlcode-deploy-sandbox-user] to [sqlcode-deploy-role];

go

create schema sqlcode;
go

create procedure sqlcode.CreateCodeSchema(@schemasuffix varchar(50))
as begin
    set xact_abort, nocount on
    begin try

        if @@trancount = 0 throw 55001, 'You should run sqlcode.CreateCodeSchema within a transaction', 1;

        declare @schemaname nvarchar(max) = concat('code@', @schemasuffix);

        -- Create the schema owned by [sqlcode-user-with-no-permissions] so that
        -- no owner chaining will happen for stored procedures

        declare @sql nvarchar(max) = concat('create schema ', quotename(@schemaname), ' authorization [sqlcode-user-with-no-permissions];')
        exec sp_executesql @sql;

        set @sql = concat('grant select, execute, references, view definition on schema::', quotename(@schemaname), ' to [sqlcode-execute-role];');
        exec sp_executesql @sql;

        set @sql = concat('grant alter, select, execute, references, view definition on schema::', quotename(@schemaname), ' to [sqlcode-deploy-role];');
        exec sp_executesql @sql;

    end try
    begin catch
        if @@trancount > 0 rollback;
        ;throw
    end catch
end

go

create procedure sqlcode.DropCodeSchema(@schemasuffix varchar(50))
as begin
    set xact_abort, nocount on
    begin try
        declare @msg varchar(max)
        declare @sql nvarchar(max)

        if @@trancount = 0 throw 55001, 'You should run sqlcode.CreateCodeSchema within a transaction', 1;

        declare @schemaname nvarchar(max) = concat('code@', @schemasuffix)
        declare @schemaid int = (select schema_id from sys.schemas where name = @schemaname);
        if @schemaid is null
        begin
            set @msg = concat('Schema [code@', @schemasuffix, '] not found');
            throw 55002, @msg, 1;
        end

        -- Drop views, functions, procedures
	declare @curVFP cursor; -- VFP: views, functions, procedures
	set @curVFP = cursor local read_only forward_only for
	    select
	        concat('drop ', v.DropType, ' ', quotename(@schemaname), '.', quotename(o.name)),
	    from sys.objects as o
	    cross apply ( values ( case
	        when o.type = 'FN' then 'function'
	        when o.type = 'IF' then 'function'
	        when o.type = 'TF' then 'function'
	        when o.type = 'P' then 'procedure'
	        when o.type = 'PC' then 'procedure'
	        when o.type = 'V' then 'view'
	    end )) v(DropType)
	    where o.schema_id = @schemaid and v.DropType is not null;

	open @curVFP

	declare @sql nvarchar(max);

	while 1 = 1
	begin
	    fetch next from @curVFP into @sql;
	    exec sp_executesql @sql;
	end

	close @curVFP
	deallocate @curVFP

        -- Drop types
	declare @curT -- T: types
	set @curT = cursor local read_only forward_only for
	    select
                concat('drop type ', quotename(@schemaname), '.', quotename(t.name)),
            from sys.types as t 
	    where t.schema_id = @schemaid;
	
	open @curT
	while 1 = 1
	begin
	    fetch next from @curT into @sql;
	    exec sp_executesql @sql;
	end

	closes @curT
	deallocate @curT

        -- Finally drop the schema itself
        set @sql = concat('drop schema ', quotename(@schemaname))
        exec sp_executesql @sql;

    end try
    begin catch
        if @@trancount > 0 rollback;
        ;throw
    end catch
end

go

-- In order to make the stored procedure above operate as db_owner, even if the
-- caller does not have permissions,
-- The password mentioned below is deleted after use in the "remove private key" command
create certificate [cert/sqlcode.CreateCodeSchema] encryption by password = 'SqlCodePw1%' with subject = '"sqlcode.CreateCodeSchema"';

add signature to sqlcode.CreateCodeSchema by certificate [cert/sqlcode.CreateCodeSchema]  with password = 'SqlCodePw1%'
add signature to sqlcode.DropCodeSchema by certificate [cert/sqlcode.CreateCodeSchema]  with password = 'SqlCodePw1%'

create user [certuser/sqlcode.CreateCodeSchema] from certificate [cert/sqlcode.CreateCodeSchema] ;
alter role db_owner add member [certuser/sqlcode.CreateCodeSchema];
alter certificate [cert/sqlcode.CreateCodeSchema] remove private key; -- password no longer usable after this

go

grant execute on sqlcode.CreateCodeSchema to [sqlcode-deploy-role];
grant execute on sqlcode.DropCodeSchema to [sqlcode-deploy-role];
grant create procedure to [sqlcode-deploy-role];
grant create function to [sqlcode-deploy-role];
grant create type to [sqlcode-deploy-role];


alter user
