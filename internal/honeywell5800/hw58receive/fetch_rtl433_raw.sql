SELECT
	id,
	time,
	json_remove(data,
		-- misinterpreted by rtl_433
		'$.state',
		-- redundant with event
		'$.heartbeat'
	) AS data
FROM rtl433_raw
WHERE model='Honeywell-Security'
	AND id>@last
	AND id<=@max
ORDER BY id ASC
LIMIT 100
