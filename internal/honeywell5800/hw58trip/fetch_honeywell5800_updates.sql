SELECT
	id,
	sensor,
	event
FROM honeywell5800_updates
WHERE id>@last
	AND id<=@max
ORDER BY id ASC
LIMIT 100
