-- Cards are inserted and moved constantly; refresh planner statistics after
-- 2 % of the table changes instead of the default 10 %, so that search keeps
-- choosing the trigram indexes after bulk imports.

-- +goose Up
ALTER TABLE cards SET (autovacuum_analyze_scale_factor = 0.02, autovacuum_vacuum_scale_factor = 0.05);

-- +goose Down
ALTER TABLE cards RESET (autovacuum_analyze_scale_factor, autovacuum_vacuum_scale_factor);
