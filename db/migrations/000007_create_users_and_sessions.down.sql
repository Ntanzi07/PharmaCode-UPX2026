ALTER TABLE summaries
    DROP COLUMN IF EXISTS reviewed_by_user_id;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
