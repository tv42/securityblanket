INSERT INTO rtl433_raw(time, freqMHz, model, data)
	SELECT @time AS new_time,
		@freqMHz AS new_freqMHz,
		json_extract(@data, '$.model') AS new_model,
		-- remove inconvenient and redundant fields
		json_remove(@data,
			-- hard-to-parse format; use a clock in our
			-- process instead
			'$.time',
			-- extracted to standalone columns
			'$.model'
		) AS new_data
		-- Deduplicate identical (after pruning) transmissions
		-- within dedup window.
		--
		-- If repeating bursts from two sensors using the same
		-- protocol (rtl_433 "model") get interleaved, this
		-- won't deduplicate them, as we don't peek inside the
		-- protocol and just see a different payload every
		-- time. That should be rare enough to not matter, and
		-- will be deduplicated by the protocol-specific
		-- logic.
		WHERE new_data
		NOT IN (
			SELECT data
			FROM rtl433_raw
			WHERE time>=@dedupTime
			AND freqMHz=new_freqMHz
			AND model=new_model
			ORDER BY id DESC
			LIMIT 1
		)
