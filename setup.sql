DROP DATABASE IF EXISTS forum WITH (FORCE);

CREATE DATABASE forum;

\c forum 

CREATE TYPE app_language AS ENUM ('en', 'id', 'ja');
CREATE TYPE app_theme AS ENUM ('light', 'dark', 'auto');
CREATE TYPE notification_type AS ENUM ('admin', 'thread', 'dm');

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE user_profiles (
    id UUID PRIMARY KEY,
    profile_picture_path TEXT NOT NULL DEFAULT '/public/pf_placeholder.png',
    biodata TEXT NOT NULL DEFAULT 'Hello, I''m using 1chan!',
    email TEXT,
    post_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    preferred_lang app_language NOT NULL DEFAULT 'en',
    preferred_theme app_theme NOT NULL DEFAULT 'auto',
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    related_id BIGINT,
    type notification_type NOT NULL,
    message TEXT,
    creation_date TIMESTAMPTZ DEFAULT NOW(),
    acknowledged_date TIMESTAMPTZ,
    FOREIGN KEY (user_id) REFERENCES users(id),
);

-- Authorisation

CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    device_id TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expiration_date TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Channels
CREATE TABLE channels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT 'Welcome to this channel!',
    channel_picture_path TEXT NOT NULL DEFAULT '/public/ch_placeholder.png'
);

-- Threads
CREATE TABLE threads (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    channel_id BIGINT,
    title TEXT NOT NULL,
    original_post TEXT NOT NULL,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP,
    last_comment_date TIMESTAMP,
    like_count INT NOT NULL DEFAULT 0,
    dislike_count INT NOT NULL DEFAULT 0,
    view_count INT NOT NULL DEFAULT 0,
    search_vector tsvector NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (channel_id) REFERENCES channels(id)
);

-- Comments
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    thread_id BIGINT NOT NULL,
    user_id UUID NOT NULL,
    comment TEXT NOT NULL,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP,
    like_count INT NOT NULL DEFAULT 0,
    dislike_count INT NOT NULL DEFAULT 0,
    search_vector tsvector NOT NULL,
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX comments_search_vector_idx ON comments USING gin(search_vector);

CREATE OR REPLACE FUNCTION update_comment_search_vector()
    RETURNS TRIGGER AS $$
    BEGIN
        NEW.search_vector := to_tsvector('english', NEW.comment);
        RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_comment_search_vector
    BEFORE INSERT OR UPDATE ON comments
    FOR EACH ROW
EXECUTE FUNCTION update_comment_search_vector();

CREATE OR REPLACE FUNCTION update_user_comment_count()
    RETURNS TRIGGER AS $$
    BEGIN
        IF (TG_OP) = 'INSERT' THEN
            UPDATE user_profiles
            SET comment_count = comment_count + 1
            WHERE id = NEW.user_id;
        ELSEIF (TG_OP = 'DELETE') THEN
            UPDATE user_profiles
            SET comment_count = comment_count - 1
            WHERE id = NEW.user_id;
        END IF;
        RETURN NULL;
    END;
    $$ LANGUAGE plpgsql;

CREATE TRIGGER user_comment_count_update
    AFTER INSERT OR DELETE
    ON comments
    FOR EACH ROW
EXECUTE FUNCTION update_user_comment_count();

-- User Auxiliary Tables
CREATE TABLE user_search_history(
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    query TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE user_channel_follows(
    user_id UUID,
    channel_id BIGINT,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (channel_id) REFERENCES channels(id),
    PRIMARY KEY (user_id, channel_id)
);

CREATE TABLE user_comment_likes (
    user_id UUID,
    comment_id BIGINT,
    variant SMALLINT NOT NULL CHECK (variant IN (0, 1)),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (comment_id) REFERENCES comments(id),
    PRIMARY KEY (user_id, comment_id)
);

CREATE TABLE user_thread_likes (
    user_id UUID,
    thread_id BIGINT,
    variant SMALLINT NOT NULL CHECK (variant IN (0, 1)),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX threads_search_vector_idx ON threads USING gin(search_vector);

CREATE OR REPLACE FUNCTION update_thread_search_vector()
RETURNS TRIGGER AS $$
    BEGIN
        NEW.search_vector := to_tsvector('english', NEW.title || ' ' || NEW.original_post);
        RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_thread_search_vector
    BEFORE INSERT OR UPDATE ON threads
    FOR EACH ROW
    EXECUTE FUNCTION update_thread_search_vector();

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    tag TEXT NOT NULL
);

CREATE OR REPLACE FUNCTION update_user_post_count()
    RETURNS TRIGGER AS $$
    BEGIN
        IF (TG_OP) = 'INSERT' THEN
            UPDATE user_profiles
            SET post_count = post_count + 1
            WHERE id = NEW.user_id;
        ELSEIF (TG_OP = 'DELETE') THEN
            UPDATE user_profiles
            SET post_count = post_count - 1
            WHERE id = NEW.user_id;
        END IF;
        RETURN NULL;
    END;
    $$ LANGUAGE plpgsql;

CREATE TRIGGER user_post_count_update
    AFTER INSERT OR DELETE
    ON threads
    FOR EACH ROW
EXECUTE FUNCTION update_user_post_count();

CREATE TABLE thread_tags (
    thread_id BIGINT,
    tag_id INT,
    PRIMARY KEY (thread_id, tag_id),
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE thread_channels (
     thread_id BIGINT,
     channel_id BIGINT,
     PRIMARY KEY (thread_id, channel_id),
     FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
     FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION update_thread_last_comment_timestamp()
RETURNS TRIGGER AS $$
    BEGIN
        UPDATE threads
        SET last_comment_date = CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
        WHERE threads.id = NEW.thread_id;
        RETURN NEW;
    end;
    $$ LANGUAGE plpgsql;

CREATE TRIGGER set_thread_last_comment_timestamp
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_thread_last_comment_timestamp();