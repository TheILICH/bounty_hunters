-- +goose Up
CREATE TABLE criminals (
    id                     BIGSERIAL PRIMARY KEY,
    name                   TEXT,
    nickname               TEXT    NOT NULL,
    reward                 INTEGER NOT NULL CHECK (reward >= 0),
    last_known_coordinates TEXT,
    difficulty             INTEGER NOT NULL CHECK (difficulty BETWEEN 1 AND 10),
    armed                  BOOLEAN NOT NULL DEFAULT false,
    has_crew               BOOLEAN NOT NULL DEFAULT false,
    escape_master          BOOLEAN NOT NULL DEFAULT false
);

-- one criminal -> many hunt logs
CREATE TABLE hunt_logs (
    id          BIGSERIAL PRIMARY KEY,
    criminal_id BIGINT      NOT NULL REFERENCES criminals (id) ON DELETE CASCADE,
    success     BOOLEAN     NOT NULL DEFAULT false,
    notes       TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX hunt_logs_criminal_id_idx ON hunt_logs (criminal_id);

-- seed: 5 criminals, 10 hunt logs (tables are new, so ids are 1..5)
INSERT INTO criminals (name, nickname, reward, last_known_coordinates, difficulty, armed, has_crew, escape_master) VALUES
    ('Vex Morran',   'Red Comet', 12000, 'Sector 7G, near the Ceres belt', 8, true,  true,  true),
    (NULL,           'Ghost',      4500, NULL,                             6, false, false, true),
    ('Dara Quill',   'Titan Fox',  8000, 'Titan orbit, dock 14',           7, true,  false, true),
    ('Brom Kessler', 'Iron Jaw',  15000, '41.2N 12.8E, Mars outpost',      9, true,  true,  false),
    ('Lio Petch',    'Small Fry',  1500, 'Europa market district',         2, false, false, false);

INSERT INTO hunt_logs (criminal_id, success, notes, created_at) VALUES
    (1, false, 'Red Comet jumped to hyperspace before we docked.',          now() - interval '20 days'),
    (1, false, 'Crew ambushed us in the Ceres belt, hull damaged.',         now() - interval '12 days'),
    (2, false, 'Ghost was seen on Phobos, gone again in one hour.',         now() - interval '18 days'),
    (2, false, 'Fake signal, Ghost was never there.',                       now() - interval '6 days'),
    (3, false, 'Target escaped through an asteroid field near Titan.',      now() - interval '15 days'),
    (3, true,  'Caught at dock 14, but she broke out of the cell later.',   now() - interval '4 days'),
    (4, false, 'Iron Jaw crew outnumbered us 6 to 2, retreated.',           now() - interval '10 days'),
    (4, false, 'Tracked his ship to Mars outpost, too many guards.',        now() - interval '3 days'),
    (5, true,  'Small Fry gave up without a fight at the Europa market.',   now() - interval '8 days'),
    (5, true,  'Caught again after his release, same market.',              now() - interval '1 day');

-- +goose Down
DROP TABLE hunt_logs;
DROP TABLE criminals;
