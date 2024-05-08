-- We want to re-create the procedure DropCodeSchema.
-- Frist, everything related to this procedure must be dropped
-- before it is re-created in the end.

drop procedure sqlcode.DropCodeSchema;

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
	        concat('drop ', v.DropType, ' ', quotename(@schemaname), '.', quotename(o.name))
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

	while 1 = 1
	begin
	    fetch next from @curVFP into @sql;
	    exec sp_executesql @sql;
	end

	close @curVFP
	deallocate @curVFP

        -- Drop types
	declare @curT cursor -- T: types
	set @curT = cursor local read_only forward_only for
	    select
                concat('drop type ', quotename(@schemaname), '.', quotename(t.name))
            from sys.types as t
	    where t.schema_id = @schemaid;

	open @curT
	while 1 = 1
	begin
	    fetch next from @curT into @sql;
	    exec sp_executesql @sql;
	end

	close @curT
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

create certificate [cert/sqlcode2] encryption by password = 'SqlCodePw1%' with subject = '"sqlcode2"';
add signature to sqlcode.DropCodeSchema by certificate [cert/sqlcode2]  with password = 'SqlCodePw1%'

create user [certuser/sqlcode2] from certificate [cert/sqlcode2] ;
alter role db_owner add member [certuser/sqlcode2];

alter certificate [cert/sqlcode2] remove private key;

grant execute on sqlcode.DropCodeSchema to [sqlcode-deploy-role];
