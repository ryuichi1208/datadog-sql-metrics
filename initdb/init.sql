CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    age INT NOT NULL
);

INSERT INTO users (name, age) VALUES ('Alice', 25), ('Bob', 30), ('Charlie', 22);

-- Create new tables for metrics collection

CREATE TABLE IF NOT EXISTS base_calls (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(50) NOT NULL,
    service_name VARCHAR(100) NOT NULL,
    request_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    response_time INTEGER NOT NULL,
    status_code INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS fax_logs (
    id SERIAL PRIMARY KEY,
    document_id VARCHAR(50) NOT NULL,
    sender VARCHAR(100) NOT NULL,
    recipient VARCHAR(100) NOT NULL,
    page_count INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO base_calls (event_id, service_name, response_time, status_code)
VALUES
    ('req-001', 'api-gateway', 150, 200),
    ('req-002', 'auth-service', 320, 200),
    ('req-003', 'user-service', 89, 200),
    ('req-004', 'payment-service', 432, 500),
    ('req-005', 'notification-service', 76, 200);

INSERT INTO fax_logs (document_id, sender, recipient, page_count, status)
VALUES
    ('doc-001', 'sales@company.com', 'client1@example.com', 3, 'sent'),
    ('doc-002', 'support@company.com', 'client2@example.com', 1, 'sent'),
    ('doc-003', 'finance@company.com', 'vendor@example.com', 5, 'failed'),
    ('doc-004', 'hr@company.com', 'candidate@example.com', 2, 'sent'),
    ('doc-005', 'marketing@company.com', 'partner@example.com', 8, 'pending');
