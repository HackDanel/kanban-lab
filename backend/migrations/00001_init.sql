-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext NOT NULL UNIQUE,
    name          text NOT NULL,
    password_hash text NOT NULL,
    role          text NOT NULL DEFAULT 'developer' CHECK (role IN ('admin', 'manager', 'developer')),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- все токены, полученные ротацией из одного входа; отзываются вместе при повторном использовании
    family_id  uuid NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);

CREATE TABLE projects (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key         text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    owner_id    uuid NOT NULL REFERENCES users (id),
    card_seq    integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE project_members (
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);
CREATE INDEX project_members_user_idx ON project_members (user_id);

CREATE TABLE boards (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX boards_project_idx ON boards (project_id, created_at);

CREATE TABLE board_columns (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id   uuid NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    name       text NOT NULL,
    position   double precision NOT NULL,
    wip_limit  integer CHECK (wip_limit > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX board_columns_board_idx ON board_columns (board_id, position);

CREATE TABLE labels (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       text NOT NULL,
    color      text NOT NULL,
    UNIQUE (project_id, name)
);

CREATE TABLE cards (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    board_id    uuid NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    column_id   uuid NOT NULL REFERENCES board_columns (id),
    number      integer NOT NULL,
    title       text NOT NULL,
    description text NOT NULL DEFAULT '',
    priority    text NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    position    double precision NOT NULL,
    assignee_id uuid REFERENCES users (id) ON DELETE SET NULL,
    reporter_id uuid NOT NULL REFERENCES users (id),
    due_date    date,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, number)
);
CREATE INDEX cards_board_order_idx ON cards (board_id, column_id, position);
CREATE INDEX cards_column_idx ON cards (column_id, position);
CREATE INDEX cards_assignee_idx ON cards (assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX cards_board_priority_idx ON cards (board_id, priority);
CREATE INDEX cards_title_trgm_idx ON cards USING gin (title gin_trgm_ops);
CREATE INDEX cards_description_trgm_idx ON cards USING gin (description gin_trgm_ops);

CREATE TABLE card_labels (
    card_id  uuid NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    label_id uuid NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, label_id)
);
CREATE INDEX card_labels_label_idx ON card_labels (label_id);

CREATE TABLE comments (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id    uuid NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    author_id  uuid NOT NULL REFERENCES users (id),
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX comments_card_idx ON comments (card_id, created_at);

CREATE TABLE attachments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id      uuid NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    uploader_id  uuid NOT NULL REFERENCES users (id),
    filename     text NOT NULL,
    content_type text NOT NULL,
    size         bigint NOT NULL,
    storage_key  text NOT NULL UNIQUE,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX attachments_card_idx ON attachments (card_id, created_at);

CREATE TABLE card_activity (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id    uuid NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    actor_id   uuid NOT NULL REFERENCES users (id),
    action     text NOT NULL,
    field      text,
    old_value  text,
    new_value  text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX card_activity_card_idx ON card_activity (card_id, created_at DESC);

-- Карточка в том виде, в каком её отдаёт API: с метками и счётчиками.
CREATE VIEW card_views AS
SELECT c.id,
       c.project_id,
       c.board_id,
       c.column_id,
       c.number,
       c.title,
       c.description,
       c.priority,
       c.position,
       c.assignee_id,
       c.reporter_id,
       c.due_date,
       c.created_at,
       c.updated_at,
       COALESCE((SELECT array_agg(cl.label_id ORDER BY cl.label_id)
                 FROM card_labels cl
                 WHERE cl.card_id = c.id), '{}')::uuid[]                        AS label_ids,
       (SELECT count(*) FROM comments m WHERE m.card_id = c.id)::integer       AS comment_count,
       (SELECT count(*) FROM attachments a WHERE a.card_id = c.id)::integer    AS attachment_count
FROM cards c;

-- +goose Down
DROP VIEW card_views;
DROP TABLE card_activity;
DROP TABLE attachments;
DROP TABLE comments;
DROP TABLE card_labels;
DROP TABLE cards;
DROP TABLE labels;
DROP TABLE board_columns;
DROP TABLE boards;
DROP TABLE project_members;
DROP TABLE projects;
DROP TABLE refresh_tokens;
DROP TABLE users;
