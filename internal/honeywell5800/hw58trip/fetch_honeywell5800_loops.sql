WITH allLoops (loop) AS (
	VALUES (1), (2), (3), (4)
)
SELECT allLoops.loop AS loop,
	honeywell5800_models.id AS model,
	honeywell5800_sensors.description AS description,
	coalesce(honeywell5800_site_loops.kind, honeywell5800_model_loops.kind) AS kind,
	coalesce(siteLabel, factoryLabel) AS label,
	coalesce(siteNormallyOpen, factoryNormallyOpen, false) AS normallyOpen
	FROM honeywell5800_sensors
	JOIN honeywell5800_models
	ON (honeywell5800_sensors.model=honeywell5800_models.id)
	JOIN allLoops
	LEFT JOIN honeywell5800_model_loops
	USING (model, loop)
	LEFT JOIN honeywell5800_site_loops
	ON (honeywell5800_site_loops.sensor=honeywell5800_sensors.id
		AND honeywell5800_site_loops.loop=allLoops.loop
	)
	WHERE honeywell5800_sensors.id=@sensor
		AND NOT coalesce(honeywell5800_site_loops.disabled, false)
		AND (NOT honeywell5800_model_loops.typicallyUnused
			OR honeywell5800_site_loops.loop IS NOT NULL)
	ORDER BY loop ASC
