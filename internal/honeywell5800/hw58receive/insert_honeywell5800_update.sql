INSERT INTO honeywell5800_updates(time, channel, sensor, event)
	SELECT @time AS new_time,
		@channel AS new_channel,
		@sensor AS new_sensor,
		@event AS new_event
		-- do not insert a new record if last time sensor was
		-- seen was within dedup time and contained the same
		-- information
		WHERE new_event NOT IN (
			SELECT event FROM honeywell5800_updates
				WHERE time>=@dedupTime
				AND channel=new_channel
				AND sensor=new_sensor
				ORDER BY id DESC
				LIMIT 1
		)
