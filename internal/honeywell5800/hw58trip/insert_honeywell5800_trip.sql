INSERT INTO honeywell5800_trips(sensor, loop, trippedBy)
	SELECT @sensor as new_sensor,
		@loop as new_loop,
		@trippedBy as new_trippedBy
		-- do not insert a new record if last trip contained
		-- the same information
		WHERE new_trippedBy NOT IN (
			SELECT trippedBy FROM honeywell5800_trips
				WHERE sensor=new_sensor
				AND loop=new_loop
				ORDER BY id DESC
				LIMIT 1
		)
