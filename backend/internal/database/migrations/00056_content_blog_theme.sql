-- +goose Up
-- The CSS of a project's own blog, pasted or uploaded in the Content Editor.
-- It only styles the article preview frame and exported .html files. Empty
-- means the plain Uplumo preview, so a removed theme deletes its row.
CREATE TABLE go_content_blog_themes (
    project_id text PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    css text NOT NULL CHECK (css <> '' AND octet_length(css) <= 102400),
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE go_content_blog_themes ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_content_blog_themes FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_content_blog_themes FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_content_blog_themes;
