DROP INDEX IF EXISTS settlements_active_group_history_idx;
DROP INDEX IF EXISTS expenses_active_group_history_idx;

ALTER TABLE settlements
    DROP CONSTRAINT IF EXISTS settlements_deletion_metadata_check,
    DROP COLUMN IF EXISTS deleted_by_user_id,
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE expenses
    DROP CONSTRAINT IF EXISTS expenses_deletion_metadata_check,
    DROP COLUMN IF EXISTS deleted_by_user_id,
    DROP COLUMN IF EXISTS deleted_at;

