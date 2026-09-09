DO $$
DECLARE
    owner_id UUID;
    member_id UUID;
    group_id UUID;
    expense_id UUID;
    invalid_expense_id UUID;
BEGIN
    INSERT INTO users (email, display_name, password_hash)
    VALUES ('owner@example.com', 'Owner', 'hash')
    RETURNING id INTO owner_id;

    INSERT INTO users (email, display_name, password_hash)
    VALUES ('member@example.com', 'Member', 'hash')
    RETURNING id INTO member_id;

    INSERT INTO groups (name, created_by_user_id)
    VALUES ('Trip', owner_id)
    RETURNING id INTO group_id;

    INSERT INTO group_members (group_id, user_id)
    VALUES (group_id, owner_id), (group_id, member_id);

    INSERT INTO expenses (group_id, paid_by_user_id, created_by_user_id, description, amount_cents)
    VALUES (group_id, owner_id, owner_id, 'Dinner', 1000)
    RETURNING id INTO expense_id;

    INSERT INTO expense_splits (expense_id, group_id, user_id, amount_cents)
    VALUES (expense_id, group_id, owner_id, 400), (expense_id, group_id, member_id, 600);

    SET CONSTRAINTS ALL IMMEDIATE;

    BEGIN
        INSERT INTO expenses (group_id, paid_by_user_id, created_by_user_id, description, amount_cents)
        VALUES (group_id, owner_id, owner_id, 'Invalid split', 1000)
        RETURNING id INTO invalid_expense_id;
        RAISE EXCEPTION 'an expense without matching splits should be rejected';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END;
$$;

SELECT count(*) AS valid_expenses FROM expenses;
