-- +goose Up
-- Uplumo uses Go sessions and scoped server queries, never Supabase client
-- access to these tables. RLS also protects accidentally restored DML grants;
-- revocation is needed separately because RLS does not protect TRUNCATE.
-- +goose StatementBegin
DO $$
DECLARE
    relation record;
    client_role text;
    schema_name text := current_schema();
BEGIN
    FOR relation IN
        SELECT c.relname, c.relkind FROM pg_class c
        WHERE c.relnamespace = current_schema()::regnamespace
          AND c.relkind IN ('r', 'p', 'v', 'm', 'S')
          AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_class'::regclass AND d.objid=c.oid AND d.deptype='e')
    LOOP
        IF relation.relkind IN ('r', 'p') THEN
            EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', schema_name, relation.relname);
        END IF;
        IF relation.relkind = 'S' THEN
            EXECUTE format('REVOKE ALL ON SEQUENCE %I.%I FROM PUBLIC', schema_name, relation.relname);
        ELSE
            EXECUTE format('REVOKE ALL ON TABLE %I.%I FROM PUBLIC', schema_name, relation.relname);
        END IF;
        FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
            IF relation.relkind = 'S' THEN
                EXECUTE format('REVOKE ALL ON SEQUENCE %I.%I FROM %I', schema_name, relation.relname, client_role);
            ELSE
                EXECUTE format('REVOKE ALL ON TABLE %I.%I FROM %I', schema_name, relation.relname, client_role);
            END IF;
        END LOOP;
    END LOOP;
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE ALL ON TABLES FROM PUBLIC', schema_name);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE ALL ON SEQUENCES FROM PUBLIC', schema_name);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC', schema_name);
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE ALL ON TABLES FROM %I', schema_name, client_role);
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE ALL ON SEQUENCES FROM %I', schema_name, client_role);
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I REVOKE EXECUTE ON FUNCTIONS FROM %I', schema_name, client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Security hardening is deliberately retained on application rollback.
SELECT 1;
