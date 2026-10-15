-- Card numbers come from one sequence per project instead of a counter column.
-- Incrementing the column serialised every card creation in a project on one
-- row and filled it with dead row versions; nextval takes no row lock.

-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    p record;
BEGIN
    FOR p IN SELECT id, card_seq FROM projects LOOP
        EXECUTE format('CREATE SEQUENCE %I START %s', 'card_seq_' || replace(p.id::text, '-', ''), p.card_seq + 1);
    END LOOP;
END
$$;
-- +goose StatementEnd
ALTER TABLE projects DROP COLUMN card_seq;

-- +goose Down
ALTER TABLE projects ADD COLUMN card_seq integer NOT NULL DEFAULT 0;
-- +goose StatementBegin
DO $$
DECLARE
    p record;
BEGIN
    FOR p IN SELECT id FROM projects LOOP
        UPDATE projects SET card_seq = COALESCE((SELECT max(number) FROM cards WHERE project_id = p.id), 0) WHERE id = p.id;
        EXECUTE format('DROP SEQUENCE IF EXISTS %I', 'card_seq_' || replace(p.id::text, '-', ''));
    END LOOP;
END
$$;
-- +goose StatementEnd
