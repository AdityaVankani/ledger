CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE CHECK (email = lower(email) AND length(email) <= 320),
    display_name TEXT NOT NULL CHECK (length(trim(display_name)) BETWEEN 1 AND 100),
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 100),
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE group_members (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    paid_by_user_id UUID NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    description TEXT NOT NULL CHECK (length(trim(description)) BETWEEN 1 AND 500),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency CHAR(3) NOT NULL DEFAULT 'INR' CHECK (currency = upper(currency)),
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, id),
    FOREIGN KEY (group_id, paid_by_user_id)
        REFERENCES group_members(group_id, user_id) ON DELETE RESTRICT,
    FOREIGN KEY (group_id, created_by_user_id)
        REFERENCES group_members(group_id, user_id) ON DELETE RESTRICT
);

CREATE TABLE expense_splits (
    expense_id UUID NOT NULL,
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (expense_id, user_id),
    FOREIGN KEY (group_id, expense_id)
        REFERENCES expenses(group_id, id) ON DELETE CASCADE,
    FOREIGN KEY (group_id, user_id)
        REFERENCES group_members(group_id, user_id) ON DELETE RESTRICT
);

CREATE TABLE settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    paid_by_user_id UUID NOT NULL,
    received_by_user_id UUID NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency CHAR(3) NOT NULL DEFAULT 'INR' CHECK (currency = upper(currency)),
    settled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    note TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (paid_by_user_id <> received_by_user_id),
    FOREIGN KEY (group_id, paid_by_user_id)
        REFERENCES group_members(group_id, user_id) ON DELETE RESTRICT,
    FOREIGN KEY (group_id, received_by_user_id)
        REFERENCES group_members(group_id, user_id) ON DELETE RESTRICT
);

CREATE INDEX expenses_group_id_expense_date_idx ON expenses (group_id, expense_date DESC, created_at DESC);
CREATE INDEX expense_splits_group_id_user_id_idx ON expense_splits (group_id, user_id);
CREATE INDEX settlements_group_id_settled_at_idx ON settlements (group_id, settled_at DESC);

CREATE FUNCTION set_updated_at() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER groups_set_updated_at BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER expenses_set_updated_at BEFORE UPDATE ON expenses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Expense writes must occur in a single transaction. At commit, every expense
-- must have splits whose total exactly matches its amount (integer cents avoids rounding errors).
CREATE FUNCTION assert_expense_split_total(target_expense_id UUID) RETURNS VOID LANGUAGE plpgsql AS $$
DECLARE
    expected_amount BIGINT;
    split_total BIGINT;
BEGIN
    SELECT amount_cents INTO expected_amount FROM expenses WHERE id = target_expense_id;
    IF NOT FOUND THEN
        RETURN;
    END IF;

    SELECT COALESCE(SUM(amount_cents), 0) INTO split_total
    FROM expense_splits WHERE expense_id = target_expense_id;

    IF split_total <> expected_amount THEN
        RAISE EXCEPTION 'expense splits (%) must equal expense amount (%)', split_total, expected_amount
            USING ERRCODE = 'check_violation';
    END IF;
END;
$$;

CREATE FUNCTION validate_expense_split_total_from_expense() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    PERFORM assert_expense_split_total(NEW.id);
    RETURN NULL;
END;
$$;

CREATE FUNCTION validate_expense_split_total_from_split() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.expense_id <> NEW.expense_id THEN
        PERFORM assert_expense_split_total(OLD.expense_id);
    END IF;
    PERFORM assert_expense_split_total(COALESCE(NEW.expense_id, OLD.expense_id));
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER expenses_validate_split_total
    AFTER INSERT OR UPDATE OF amount_cents ON expenses
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
    EXECUTE FUNCTION validate_expense_split_total_from_expense();
CREATE CONSTRAINT TRIGGER expense_splits_validate_total
    AFTER INSERT OR UPDATE OR DELETE ON expense_splits
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
    EXECUTE FUNCTION validate_expense_split_total_from_split();
