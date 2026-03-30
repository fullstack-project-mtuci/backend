-- Recreate schema per expanded specification.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Departments
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Projects
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID REFERENCES departments(id),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Expense categories
CREATE TABLE IF NOT EXISTS expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Currency reference
CREATE TABLE IF NOT EXISTS currencies (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Users adjustments
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id),
    ADD COLUMN IF NOT EXISTS manager_id UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Budgets
CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('department', 'project')),
    scope_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    total_limit NUMERIC(18,2) NOT NULL,
    reserved_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    spent_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_budgets_scope ON budgets(scope_type, scope_id);

-- Trip requests
DROP TABLE IF EXISTS trip_requests CASCADE;
CREATE TABLE trip_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES users(id),
    project_id UUID REFERENCES projects(id),
    budget_id UUID REFERENCES budgets(id),
    destination_city TEXT NOT NULL,
    destination_country TEXT NOT NULL,
    purpose TEXT NOT NULL,
    comment TEXT,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    planned_transport NUMERIC(18,2) NOT NULL DEFAULT 0,
    planned_hotel NUMERIC(18,2) NOT NULL DEFAULT 0,
    planned_daily_allowance NUMERIC(18,2) NOT NULL DEFAULT 0,
    planned_other NUMERIC(18,2) NOT NULL DEFAULT 0,
    planned_total NUMERIC(18,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    submitted_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_trip_requests_employee ON trip_requests(employee_id);
CREATE INDEX idx_trip_requests_project ON trip_requests(project_id);
CREATE INDEX idx_trip_requests_status ON trip_requests(status);

-- Advance requests
CREATE TABLE IF NOT EXISTS advance_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_request_id UUID NOT NULL REFERENCES trip_requests(id) ON DELETE CASCADE,
    requested_amount NUMERIC(18,2) NOT NULL,
    approved_amount NUMERIC(18,2),
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    comment TEXT,
    submitted_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Expense reports
CREATE TABLE IF NOT EXISTS expense_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_request_id UUID NOT NULL REFERENCES trip_requests(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES users(id),
    advance_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    total_expenses NUMERIC(18,2) NOT NULL DEFAULT 0,
    balance_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    submitted_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_expense_reports_trip ON expense_reports(trip_request_id);
CREATE INDEX idx_expense_reports_employee ON expense_reports(employee_id);

-- Receipt files
CREATE TABLE IF NOT EXISTS receipt_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploaded_by UUID NOT NULL REFERENCES users(id),
    storage_path TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    checksum TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Expense items
CREATE TABLE IF NOT EXISTS expense_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_report_id UUID NOT NULL REFERENCES expense_reports(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    expense_date DATE NOT NULL,
    vendor_name TEXT,
    amount NUMERIC(18,2) NOT NULL,
    currency TEXT NOT NULL,
    tax_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    description TEXT,
    receipt_file_id UUID REFERENCES receipt_files(id),
    source TEXT NOT NULL,
    ocr_confidence NUMERIC(5,2),
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_expense_items_report ON expense_items(expense_report_id);
CREATE INDEX idx_expense_items_receipt ON expense_items(receipt_file_id);

-- Receipt recognition
CREATE TABLE IF NOT EXISTS receipt_recognitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_file_id UUID NOT NULL REFERENCES receipt_files(id) ON DELETE CASCADE,
    raw_response_json JSONB,
    extracted_date DATE,
    extracted_amount NUMERIC(18,2),
    extracted_currency TEXT,
    extracted_vendor TEXT,
    extracted_tax NUMERIC(18,2),
    confidence_score NUMERIC(5,2),
    status TEXT NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Approval actions
CREATE TABLE IF NOT EXISTS approval_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    action TEXT NOT NULL,
    actor_id UUID NOT NULL REFERENCES users(id),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_approval_entity ON approval_actions(entity_type, entity_id);

-- Audit logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    before_json JSONB,
    after_json JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);
