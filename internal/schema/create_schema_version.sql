CREATE TABLE IF NOT EXISTS schema_version (
	version INT NOT NULL
);
INSERT OR IGNORE
	INTO schema_version(version)
	VALUES (0);
