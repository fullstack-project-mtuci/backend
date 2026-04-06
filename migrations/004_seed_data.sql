-- Seed reference data for demo environments.

-- Departments
INSERT INTO departments (name, code)
VALUES
    ('Engineering', 'ENG'),
    ('Sales', 'SALES'),
    ('Finance', 'FIN')
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    updated_at = now();

-- Projects mapped to departments
INSERT INTO projects (name, code, department_id, is_active)
VALUES
    ('Travel Automation Platform', 'TRAVEL-AUTO', (SELECT id FROM departments WHERE code = 'ENG'), TRUE),
    ('APAC Expansion', 'APAC-EXP', (SELECT id FROM departments WHERE code = 'SALES'), TRUE),
    ('Expense Analytics Suite', 'EXP-ANALYTICS', (SELECT id FROM departments WHERE code = 'FIN'), TRUE)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    department_id = EXCLUDED.department_id,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- Expense categories
INSERT INTO expense_categories (name, code, is_active)
VALUES
    ('Transportation', 'TRANSPORT', TRUE),
    ('Lodging', 'LODGING', TRUE),
    ('Meals', 'MEALS', TRUE),
    ('Office & Misc', 'OFFICE', TRUE)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- Currencies
INSERT INTO currencies (code, name, is_active)
VALUES
    ('RUB', 'Russian Ruble', TRUE),
    ('USD', 'US Dollar', TRUE),
    ('EUR', 'Euro', TRUE)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- Budgets per department/project (avoid duplicates if already present)
INSERT INTO budgets (scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency)
SELECT
    'department',
    d.id,
    DATE '2024-01-01',
    DATE '2024-12-31',
    1500000,
    250000,
    110000,
    'RUB'
FROM departments d
WHERE d.code = 'ENG'
  AND NOT EXISTS (
    SELECT 1 FROM budgets b
    WHERE b.scope_type = 'department'
      AND b.scope_id = d.id
      AND b.period_start = DATE '2024-01-01'
);

INSERT INTO budgets (scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency)
SELECT
    'department',
    d.id,
    DATE '2024-01-01',
    DATE '2024-12-31',
    1200000,
    180000,
    95000,
    'RUB'
FROM departments d
WHERE d.code = 'SALES'
  AND NOT EXISTS (
    SELECT 1 FROM budgets b
    WHERE b.scope_type = 'department'
      AND b.scope_id = d.id
      AND b.period_start = DATE '2024-01-01'
);

INSERT INTO budgets (scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency)
SELECT
    'project',
    p.id,
    DATE '2024-01-01',
    DATE '2024-12-31',
    600000,
    120000,
    45000,
    'RUB'
FROM projects p
WHERE p.code = 'TRAVEL-AUTO'
  AND NOT EXISTS (
    SELECT 1 FROM budgets b
    WHERE b.scope_type = 'project'
      AND b.scope_id = p.id
      AND b.period_start = DATE '2024-01-01'
);

INSERT INTO budgets (scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency)
SELECT
    'project',
    p.id,
    DATE '2024-01-01',
    DATE '2024-12-31',
    450000,
    90000,
    38000,
    'RUB'
FROM projects p
WHERE p.code = 'APAC-EXP'
  AND NOT EXISTS (
    SELECT 1 FROM budgets b
    WHERE b.scope_type = 'project'
      AND b.scope_id = p.id
      AND b.period_start = DATE '2024-01-01'
);

-- Users for each role
INSERT INTO users (email, password_hash, full_name, role, department_id, manager_id, is_active)
SELECT
    'manager@example.com',
    crypt('Manager123!', gen_salt('bf')),
    'Maria Petrova',
    'manager',
    d.id,
    NULL,
    TRUE
FROM departments d
WHERE d.code = 'ENG'
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (email, password_hash, full_name, role, department_id, manager_id, is_active)
SELECT
    'accountant@example.com',
    crypt('Accountant123!', gen_salt('bf')),
    'Andrey Sokolov',
    'accountant',
    d.id,
    NULL,
    TRUE
FROM departments d
WHERE d.code = 'FIN'
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (email, password_hash, full_name, role, department_id, manager_id, is_active)
SELECT
    'employee.alex@example.com',
    crypt('Employee123!', gen_salt('bf')),
    'Alexei Vinogradov',
    'employee',
    d.id,
    m.id,
    TRUE
FROM departments d
JOIN users m ON m.email = 'manager@example.com'
WHERE d.code = 'ENG'
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (email, password_hash, full_name, role, department_id, manager_id, is_active)
SELECT
    'employee.julia@example.com',
    crypt('Employee123!', gen_salt('bf')),
    'Julia Pavlova',
    'employee',
    d.id,
    NULL,
    TRUE
FROM departments d
WHERE d.code = 'SALES'
ON CONFLICT (email) DO NOTHING;

-- Scenario 1: approved trip with advance and expense report
WITH alex_trip AS (
    INSERT INTO trip_requests (
        employee_id,
        project_id,
        budget_id,
        destination_city,
        destination_country,
        purpose,
        comment,
        start_date,
        end_date,
        planned_transport,
        planned_hotel,
        planned_daily_allowance,
        planned_other,
        planned_total,
        currency,
        status,
        submitted_at,
        approved_at
    )
    VALUES (
        (SELECT id FROM users WHERE email = 'employee.alex@example.com'),
        (SELECT id FROM projects WHERE code = 'TRAVEL-AUTO'),
        (
            SELECT id
            FROM budgets
            WHERE scope_type = 'project'
              AND scope_id = (SELECT id FROM projects WHERE code = 'TRAVEL-AUTO')
            ORDER BY period_start DESC
            LIMIT 1
        ),
        'Berlin',
        'Germany',
        'Vendor workshops',
        'Discuss automation backlog with supplier',
        DATE '2024-05-12',
        DATE '2024-05-16',
        35000,
        42000,
        15000,
        6000,
        98000,
        'EUR',
        'manager_approved',
        now() - interval '20 days',
        now() - interval '15 days'
    )
    RETURNING id, employee_id
), alex_advance AS (
    INSERT INTO advance_requests (
        trip_request_id,
        requested_amount,
        approved_amount,
        currency,
        status,
        comment,
        submitted_at,
        processed_at
    )
    SELECT
        id,
        80000,
        80000,
        'EUR',
        'approved',
        'Advance approved for transport and hotel',
        now() - interval '19 days',
        now() - interval '17 days'
    FROM alex_trip
), alex_report AS (
    INSERT INTO expense_reports (
        trip_request_id,
        employee_id,
        advance_amount,
        total_expenses,
        balance_amount,
        currency,
        status,
        submitted_at,
        reviewed_at,
        closed_at
    )
    SELECT
        id,
        employee_id,
        80000,
        78000,
        2000,
        'EUR',
        'submitted',
        now() - interval '10 days',
        now() - interval '5 days',
        NULL
    FROM alex_trip
    RETURNING id
)
INSERT INTO expense_items (
    expense_report_id,
    category,
    expense_date,
    vendor_name,
    amount,
    currency,
    tax_amount,
    description,
    source,
    ocr_confidence,
    status
)
SELECT
    ar.id,
    v.category,
    v.expense_date,
    v.vendor_name,
    v.amount,
    v.currency,
    v.tax_amount,
    v.description,
    v.source,
    v.ocr_confidence,
    v.status
FROM alex_report ar
CROSS JOIN (
    VALUES
        ('TRANSPORT', DATE '2024-05-12', 'Lufthansa', 35000, 'EUR', 5800, 'Flights Moscow-Berlin-Moscow', 'manual', 0.98, 'accepted'),
        ('LODGING', DATE '2024-05-13', 'Hilton Berlin', 42000, 'EUR', 0, 'Hotel stay for 3 nights', 'manual', NULL, 'pending_review')
) AS v(category, expense_date, vendor_name, amount, currency, tax_amount, description, source, ocr_confidence, status);

-- Scenario 2: submitted trip waiting for approval
WITH julia_trip AS (
    INSERT INTO trip_requests (
        employee_id,
        project_id,
        budget_id,
        destination_city,
        destination_country,
        purpose,
        comment,
        start_date,
        end_date,
        planned_transport,
        planned_hotel,
        planned_daily_allowance,
        planned_other,
        planned_total,
        currency,
        status,
        submitted_at
    )
    VALUES (
        (SELECT id FROM users WHERE email = 'employee.julia@example.com'),
        (SELECT id FROM projects WHERE code = 'APAC-EXP'),
        (
            SELECT id
            FROM budgets
            WHERE scope_type = 'department'
              AND scope_id = (SELECT id FROM departments WHERE code = 'SALES')
            ORDER BY period_start DESC
            LIMIT 1
        ),
        'Tokyo',
        'Japan',
        'Client demos and sales training',
        'Need to visit APAC partners and team',
        DATE '2024-06-02',
        DATE '2024-06-08',
        28000,
        52000,
        18000,
        8000,
        106000,
        'USD',
        'submitted',
        now() - interval '4 days'
    )
    RETURNING id
)
INSERT INTO advance_requests (
    trip_request_id,
    requested_amount,
    approved_amount,
    currency,
    status,
    comment,
    submitted_at
)
SELECT
    id,
    90000,
    NULL,
    'USD',
    'submitted',
    'Awaiting manager review',
    now() - interval '3 days'
FROM julia_trip;
