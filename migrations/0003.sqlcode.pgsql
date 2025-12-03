-- ======================================================================
-- create users and roles
-- ======================================================================
do $$
begin
    -- role that will own the sqlcode schemas (actual code schemas), with no login
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-user-with-no-permissions'
    ) then
        create role "sqlcode-user-with-no-permissions" nologin;
    end if;

    -- role that owns the management schema/procedures (security definer)
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-definer-role'
    ) then
        create role "sqlcode-definer-role" nologin;
    end if;

    -- role that gets execute / usage on code schemas (for humans debugging etc.)
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-execute-role'
    ) then
        create role "sqlcode-execute-role";
    end if;

    -- role for calling createcodeschema / dropcodeschema;
    -- this role does not own the procedures, it only calls them.
    if not exists (
        select 1
        from pg_roles
        where rolname = 'sqlcode-deploy-role'
    ) then
        create role "sqlcode-deploy-role";
    end if;

    -- sandbox role used during deploys, which only has sqlcode-deploy-role
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
-- grant permissions / role memberships
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
-- create schema for management code (owner = definer role)
-- ======================================================================

do $$
begin
    if not exists (
        select 1 from pg_namespace where nspname = 'sqlcode'
    ) then
        create schema sqlcode authorization "sqlcode-definer-role";
    end if;
end;
$$;

-- ======================================================================
-- create procedures (security definer)
-- ======================================================================

create or replace procedure sqlcode.createcodeschema(schemasuffix varchar)
language plpgsql
security definer
as $$
declare
    schemaname text := format('code@%s', schemasuffix);
begin
    -- harden search_path for security-definer (optional but recommended)
    perform set_config('search_path', 'pg_catalog', true);

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

create or replace procedure sqlcode.dropcodeschema(schemasuffix varchar)
language plpgsql
security definer
as $$
declare
    schemaname     text    := format('code@%s', schemasuffix);
    schema_exists  boolean;
begin
    -- harden search_path for security-definer (optional but recommended)
    perform set_config('search_path', 'pg_catalog', true);

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

-- similar behaviour as mssql getapplock
-- PostgreSQL advisory locks are session-based by default
create or replace function sqlcode.get_applock(
    resource text,
    timeout_ms integer default 0
)
returns integer
language plpgsql
as $$
declare
    resource_key bigint;
    acquired     boolean;
    waited_ms    integer := 0;
begin
    -- convert string to advisory-lock key
    select hashtext(resource) into resource_key;

    -- attempt lock with timeout loop
    loop
        select pg_try_advisory_lock_shared(resource_key)
        into acquired;

        if acquired then
            return 1;  -- lock acquired (success)
        end if;

        if waited_ms >= timeout_ms then
            return 0;  -- timeout
        end if;

        perform pg_sleep(0.01);    -- sleep 10 ms
        waited_ms := waited_ms + 10;
    end loop;

    return null;  -- safety fallback (should never hit)
end;
$$;

create or replace function sqlcode.release_applock(resource text)
returns boolean
language sql
as $$
    select pg_advisory_unlock_shared(hashtext(resource));
$$;


-- ensure procedures are owned by the definer role
alter procedure sqlcode.createcodeschema(varchar)
    owner to "sqlcode-definer-role";

alter procedure sqlcode.dropcodeschema(varchar)
    owner to "sqlcode-definer-role";

-- ======================================================================
-- privileges on the procedures and base schema
-- ======================================================================

-- allow deploy role to call the management procedures
grant execute on procedure sqlcode.createcodeschema(varchar)
    to "sqlcode-deploy-role";

grant execute on procedure sqlcode.dropcodeschema(varchar)
    to "sqlcode-deploy-role";

-- usually deploy role does not need create in the sqlcode management schema
-- (the procedures handle creation in separate "code@..." schemas)
grant usage on schema sqlcode
    to "sqlcode-deploy-role";
