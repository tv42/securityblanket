UPDATE honeywell5800_trips
	SET clearedBy=@clearedBy
	WHERE clearedBy IS NULL
		AND id IN (
			SELECT id FROM honeywell5800_trips
				WHERE sensor=@sensor
				AND loop=@loop
				ORDER BY id DESC
				LIMIT 1
	)
