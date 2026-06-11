CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    telegram_id BIGINT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE houses (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT NOT NULL,
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE house_members (
    house_id   UUID NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
    PRIMARY KEY (house_id, user_id)
);

CREATE TYPE task_type AS ENUM ('recurring', 'one_time');
CREATE TYPE task_priority AS ENUM ('low', 'normal', 'high');
CREATE TYPE reminder_strategy AS ENUM ('simple', 'advance', 'meeting');

CREATE TABLE tasks (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    house_id            UUID NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    created_by          UUID NOT NULL REFERENCES users(id),
    assigned_to         UUID REFERENCES users(id),
    title               TEXT NOT NULL,
    task_type           task_type NOT NULL DEFAULT 'one_time',
    priority            task_priority NOT NULL DEFAULT 'normal',
    reminder_strategy   reminder_strategy NOT NULL DEFAULT 'simple',
    due_at              TIMESTAMP,
    next_run_at         TIMESTAMP,
    interval_days       INT,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE reminder_rules (
    id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id   UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    remind_at TIMESTAMP NOT NULL,
    is_sent   BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at   TIMESTAMP
);

CREATE TABLE task_history (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id      UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    completed_by UUID NOT NULL REFERENCES users(id),
    completed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_house_id ON tasks(house_id);
CREATE INDEX idx_tasks_assigned_to ON tasks(assigned_to);
CREATE INDEX idx_tasks_next_run_at ON tasks(next_run_at) WHERE is_active = TRUE;
CREATE INDEX idx_reminder_rules_remind_at ON reminder_rules(remind_at) WHERE is_sent = FALSE;
CREATE INDEX idx_task_history_task_id ON task_history(task_id);
