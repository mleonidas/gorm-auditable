# gorm-auditable
Hooks for audit table when using gorm

# audit table setup

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS audit_logs(
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
  table_name varchar,
  operation_type varchar,
  object_id varchar,
  data jsonb,
  user_id varchar,
  created_at timestamptz NOT NULL DEFAULT now()
);
```
