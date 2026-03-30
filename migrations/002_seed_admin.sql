INSERT INTO users (email, password_hash, full_name, role)
VALUES (
    'admin@example.com',
    crypt('Admin123!', gen_salt('bf')),
    'Default Admin',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
