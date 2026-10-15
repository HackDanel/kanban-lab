-- Text search is always scoped to one board. With board_id inside the trigram
-- index, PostgreSQL matches only that board's cards instead of matching across
-- the whole table and filtering afterwards.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE INDEX cards_board_title_trgm_idx ON cards USING gin (board_id, title gin_trgm_ops);
CREATE INDEX cards_board_description_trgm_idx ON cards USING gin (board_id, description gin_trgm_ops);
DROP INDEX cards_title_trgm_idx;
DROP INDEX cards_description_trgm_idx;

-- +goose Down
CREATE INDEX cards_title_trgm_idx ON cards USING gin (title gin_trgm_ops);
CREATE INDEX cards_description_trgm_idx ON cards USING gin (description gin_trgm_ops);
DROP INDEX cards_board_title_trgm_idx;
DROP INDEX cards_board_description_trgm_idx;
