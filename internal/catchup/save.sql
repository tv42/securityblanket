INSERT INTO catchup(name, last)
	VALUES (@name, @last)
	ON CONFLICT(name)
		DO UPDATE SET last=excluded.last
