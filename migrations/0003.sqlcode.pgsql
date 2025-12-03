-- ======================================================================
-- Create users and roles
-- ======================================================================
do $$
begin
    -- This role will own the sqlcode schemas, so that created functions etc.
    -- are owned by a role without permissions; this means functions/procedures
    -- will not get more permissions than the caller already has (unless you use
    -- SECURITY DEFINER somewhere).
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-user-with-no-permissions'
    ) then
        create role "sqlcode-user-with-no-permissions" nologin;
    end if;

    -- This role will be granted execute (usage) permissions on all sqlcode schemas;
    -- useful e.g. for humans logging in to debug.
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-execute-role'
    ) then
        create role "sqlcode-execute-role";
    end if;

    -- Role for calling CreateCodeSchema / DropCodeSchema; the role will also be granted
    -- control over all schemas created this way.
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-deploy-role'
    ) then
        create role "sqlcode-deploy-role";
    end if;

    -- Make a role that *only* has this deploy role. During deploys we SET ROLE to this
    -- so that we can more safely deploy code with restricted permissions.
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-deploy-sandbox-user'
    ) then
        create role "sqlcode-deploy-sandbox-user" nologin;
    end if;
end;
$$;

-- ======================================================================
-- grant permissions
-- ======================================================================

do $$
begin
    -- grant "sqlcode-deploy-role" to "sqlcode-deploy-sandbox-user"
    if not exists (
        select 1
        from pg_auth_members m
        join pg_roles r_role on r_role.oid = m.roleid
        join pg_roles r_member on r_member.oid = m.member
        where r_role.rolname = 'sqlcode-deploy-role'
          and r_member.rolname = 'sqlcode-deploy-sandbox-user'
    ) then
        grant "sqlcode-deploy-role" to "sqlcode-deploy-sandbox-user";
    end if;

end;
$$;

-- ======================================================================
-- create schema
-- ======================================================================

-- Base schema to hold the procedures etc.
do $$
begin
    if not exists (
        select 1 from pg_namespace where nspname = 'sqlcode'
    ) then
        create schema sqlcode;
    end if;
end;
$$;

-- ======================================================================
-- create procedures
-- ======================================================================

create or replace procedure sqlcode.createcodeschema(schemasuffix varchar)
language plpgsql
security definer
as $$
declare
    schemaname text := format('code@%s', schemasuffix);
begin
    -- create the schema owned by "sqlcode-user-with-no-permissions"
    execute format(
        'create schema %I authorization %I',
        schemaname,
        'sqlcode-user-with-no-permissions'
    );

    -- grant schema privileges
    execute format(
        'grant usage on schema %I to %I',
        schemaname,
        'sqlcode-execute-role'
    );

    execute format(
        'grant usage, create on schema %I to %I',
        schemaname,
        'sqlcode-deploy-role'
    );

exception
    when others then
        raise;
end;
$$;

-- ======================================================================
-- procedure: sqlcode.dropcodeschema
-- ======================================================================

create or replace procedure sqlcode.dropcodeschema(schemasuffix varchar)
language plpgsql
security definer
as $$
declare
    schemaname text := format('code@%s', schemasuffix);
    schema_exists boolean;
begin
    -- check schema existence
    select exists (
        select 1
        from pg_namespace
        where nspname = schemaname
    ) into schema_exists;

    if not schema_exists then
        raise exception 'schema "%" not found', schemaname;
    end if;

    -- drop the schema and all objects within it
    execute format('drop schema %I cascade', schemaname);

exception
    when others then
        raise;
end;
$$;

-- ======================================================================
-- privileges on the procedures and base schema
-- ======================================================================

grant execute on procedure sqlcode.createcodeschema(varchar)
    to "sqlcode-deploy-role";

grant execute on procedure sqlcode.dropcodeschema(varchar)
    to "sqlcode-deploy-role";

grant usage, create on schema sqlcode
    to "sqlcode-deploy-role";
