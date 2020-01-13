CREATE TABLE catchup (
	name TEXT NOT NULL PRIMARY KEY
		CONSTRAINT 'name is not empty' CHECK (name<>''),
	last INTEGER NOT NULL
		CONSTRAINT 'last is not negative' CHECK (last>=0)
)
	WITHOUT ROWID;

CREATE TABLE rtl433_raw (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	time TEXT NOT NULL
		DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
	freqMHz INTEGER NOT NULL
		CONSTRAINT 'freqMHz is positive' CHECK (freqMhz>0),
	model TEXT NOT NULL
		CONSTRAINT 'model is not empty' CHECK (model<>''),
	data TEXT NOT NULL
		CONSTRAINT 'data is not empty' CHECK (data<>'')
);

-- Kinds of sensor models known to the code.
--
-- Also used to prevent typos when manually editing the database
-- (assuming you have the sense to enforce foreign keys; add `PRAGMA
-- foreign_keys=1;` to `~/.sqliterc`).
CREATE TABLE honeywell5800_loop_kinds (
	id TEXT NOT NULL PRIMARY KEY
		CONSTRAINT 'id is not empty' CHECK (id<>'')
)
	WITHOUT ROWID;

INSERT INTO honeywell5800_loop_kinds(id)
	VALUES
		('door open'),
		('door or window open'),
		('glass break'),
		('heat detector'),
		('key fob button'),
		('low temperature'),
		('maintenance needed'),
		('medical alert'),
		('motion detector'),
		('panic button'),
		('smoke detector'),
		('tamper'),
		('tilt switch'),
		('window open');

-- Sensor models known to the system.
--
-- Also used to prevent typos when manually editing the database
-- (assuming you have the sense to enforce foreign keys; add `PRAGMA
-- foreign_keys=1;` to `~/.sqliterc`).
CREATE TABLE honeywell5800_models (
	id TEXT NOT NULL PRIMARY KEY
		CONSTRAINT 'id is not empty' CHECK (id<>''),
	description TEXT
)
	WITHOUT ROWID;

CREATE TABLE honeywell5800_sensors (
	id INTEGER NOT NULL PRIMARY KEY,
	created TEXT NOT NULL
		DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
	model TEXT REFERENCES honeywell5800_models(id)
		ON DELETE SET NULL,
	description TEXT NOT NULL
		DEFAULT ''
);

CREATE TABLE honeywell5800_updates (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	time TEXT NOT NULL,
	channel INTEGER NOT NULL
		CONSTRAINT 'channel value in range' CHECK (
			channel >= 0
			AND channel < 16
		),
	sensor INTEGER NOT NULL
		REFERENCES honeywell5800_sensors(id)
		ON DELETE CASCADE,
	event INTEGER NOT NULL
		CONSTRAINT 'event value in range' CHECK (
			event >= 0
			AND event < 256
		)
);

-- Loops not present in this table are not present in the hardware.
CREATE TABLE honeywell5800_model_loops (
	model TEXT NOT NULL,
	loop INTEGER NOT NULL
		CONSTRAINT 'loop value in range' CHECK (
			loop >= 1
			AND loop <= 4
		),
	kind TEXT NOT NULL REFERENCES honeywell5800_loop_kinds(id),
	factoryLabel TEXT
		CONSTRAINT 'label is not empty when present' CHECK (
			factoryLabel IS NULL OR factoryLabel<>''
		),
	factoryNormallyOpen BOOLEAN NOT NULL
		DEFAULT false,
	typicallyUnused BOOLEAN NOT NULL
		DEFAULT false,
	PRIMARY KEY (model, loop)
)
	WITHOUT ROWID;

CREATE TABLE honeywell5800_site_loops (
	sensor INTEGER NOT NULL
		REFERENCES honeywell5800_sensors(id)
		ON DELETE CASCADE,
	loop INTEGER NOT NULL
		CONSTRAINT 'loop value in range' CHECK (
			loop >= 1
			AND loop <= 4
		),
	kind TEXT REFERENCES honeywell5800_loop_kinds(id),
	siteLabel TEXT,
	siteNormallyOpen BOOLEAN,
	disabled BOOLEAN,
	PRIMARY KEY (sensor, loop)
)
	WITHOUT ROWID;

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5800MINI', 'door/window');

INSERT INTO honeywell5800_model_loops(model, loop, kind)
	VALUES
		('5800MINI', 1, 'door or window open'),
		('5800MINI', 4, 'tamper');

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5800PIR-RES', 'motion detector');

INSERT INTO honeywell5800_model_loops(model, loop, kind)
	VALUES
		('5800PIR-RES', 1, 'motion detector'),
		('5800PIR-RES', 4, 'tamper');

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5853', 'glass break');

INSERT INTO honeywell5800_model_loops(model, loop, kind)
	VALUES
		('5853', 1, 'glass break'),
		('5853', 4, 'tamper');

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5816', 'door/window');

INSERT INTO honeywell5800_model_loops(model, loop, kind, factoryLabel, typicallyUnused)
	VALUES
		-- marking wired as typicallyUnused as most
		-- installations just leave the contacts unused
		('5816', 1, 'door or window open', 'wired', true),
		('5816', 2, 'door or window open', 'magnet', false),
		('5816', 4, 'tamper', NULL, false);

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5808W3', 'smoke detector');

INSERT INTO honeywell5800_model_loops(model, loop, kind)
	VALUES
		('5808W3', 1, 'smoke detector'),
		('5808W3', 2, 'maintenance needed'),
		('5808W3', 3, 'low temperature'),
		('5808W3', 4, 'tamper');

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5818MNL', 'door/window');

INSERT INTO honeywell5800_model_loops(model, loop, kind)
	VALUES
		('5818MNL', 1, 'door or window open'),
		('5818MNL', 4, 'tamper');

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5822T', 'tilt switch');

INSERT INTO honeywell5800_model_loops(model, loop, kind, factoryLabel, typicallyUnused)
	VALUES
		-- marking wired as typicallyUnused as most
		-- installations just leave the contacts unused
		('5822T', 1, 'door or window open', 'wired', true),
		('5822T', 3, 'tilt switch', NULL, false),
		-- unconfirmed
		('5822T', 4, 'tamper', NULL, false);

INSERT INTO honeywell5800_models(id, description)
	VALUES ('5802MN', 'medical alert');

INSERT INTO honeywell5800_model_loops(model, loop, kind)
	VALUES
		('5802MN', 1, 'medical alert');

CREATE TABLE honeywell5800_trips (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	-- duplicates trippedBy.sensor but makes joins a lot simpler
	sensor INTEGER NOT NULL
		REFERENCES honeywell5800_sensors(id)
		ON DELETE CASCADE,
	loop INTEGER NOT NULL
		CONSTRAINT 'loop value in range' CHECK (
			loop >= 1
			AND loop <= 4
		),
	trippedBy INTEGER NOT NULL
		REFERENCES honeywell5800_updates(id)
		ON DELETE CASCADE,
	clearedBy INTEGER
		REFERENCES honeywell5800_updates(id)
		ON DELETE CASCADE
);
