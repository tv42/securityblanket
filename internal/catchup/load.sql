SELECT coalesce(max(last), 0) AS last
	FROM catchup
	WHERE name=@name
	LIMIT 1
