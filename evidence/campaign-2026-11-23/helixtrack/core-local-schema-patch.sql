-- LOCAL-INSTANCE PATCH (HRD-029 evidence): the shipped Application/Database/Definition.sqlite lacks columns/tables
-- that the Go handlers read/write (ticket.version, project.version, ticket_history).
ALTER TABLE ticket ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE project ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
CREATE TABLE IF NOT EXISTS ticket_history (
 id TEXT NOT NULL PRIMARY KEY UNIQUE, ticket_id TEXT NOT NULL, version INTEGER NOT NULL, action TEXT NOT NULL,
 user_id TEXT NOT NULL, timestamp INTEGER NOT NULL, old_data TEXT, new_data TEXT, change_summary TEXT, conflict_data TEXT);
