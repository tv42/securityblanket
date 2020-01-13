INSERT INTO honeywell5800_sensors(id, created)
	VALUES (@sensor, @created)
	ON CONFLICT(id) DO NOTHING
