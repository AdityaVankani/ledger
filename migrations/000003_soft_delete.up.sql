ALTER TABLE expenses
    ADD COLUMN deleted_at TIMESTAMPTZ,
    ADD COLUMN deleted_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT expenses_deletion_metadata_check CHECK (
        (deleted_at IS NULL AND deleted_by_user_id IS NULL) OR
        (deleted_at IS NOT NULL AND deleted_by_user_id IS NOT NULL)
    );

ALTER TABLE settlements
    ADD COLUMN deleted_at TIMESTAMPTZ,
    ADD COLUMN deleted_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT settlements_deletion_metadata_check CHECK (
        (deleted_at IS NULL AND deleted_by_user_id IS NULL) OR
        (deleted_at IS NOT NULL AND deleted_by_user_id IS NOT NULL)
    );

CREATE INDEX expenses_active_group_history_idx
    ON expenses (group_id, expense_date DESC, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX settlements_active_group_history_idx
    ON settlements (group_id, settled_at DESC)
    WHERE deleted_at IS NULL;

