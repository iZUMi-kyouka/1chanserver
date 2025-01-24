DROP DATABASE IF EXISTS forum WITH (FORCE);

CREATE DATABASE forum;

\c forum 

CREATE TYPE app_language AS ENUM ('en', 'id', 'ja');
CREATE TYPE app_theme AS ENUM ('light', 'dark', 'auto');
CREATE TYPE notification_type AS ENUM ('admin', 'thread', 'dm');
CREATE TYPE report_status AS ENUM ('pending', 'resolved');

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE user_profiles (
    id UUID PRIMARY KEY,
    profile_picture_path TEXT,
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
    FOREIGN KEY (user_id) REFERENCES users(id)
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
    channel_picture_path TEXT
);

-- Threads
CREATE TABLE threads (
    id BIGSERIAL PRIMARY KEY ,
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
    comment_count INT NOT NULL DEFAULT 0,
    search_vector tsvector NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (channel_id) REFERENCES channels(id)
);

-- Comments
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    thread_id INT NOT NULL,
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

-- Polls
CREATE TABLE polls (
    id BIGSERIAL PRIMARY KEY,
    thread_id INT NOT NULL,
    question TEXT NOT NULL,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    max_choice INT NOT NULL DEFAULT 1,
    FOREIGN KEY (thread_id) REFERENCES threads(id)
);

CREATE TABLE poll_options (
    poll_id BIGINT NOT NULL,
    option_id INT NOT NULL,
    option_text TEXT NOT NULL,
    vote_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (poll_id, option_id),
    FOREIGN KEY (poll_id) REFERENCES polls(id)
);

CREATE TABLE user_poll_votes (
    user_id UUID NOT NULL,
    poll_id BIGINT NOT NULL,
    poll_option_id INT NOT NULL,
    PRIMARY KEY (user_id, poll_id, poll_option_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (poll_id) REFERENCES polls(id),
    FOREIGN KEY (poll_id, poll_option_id) REFERENCES poll_options(poll_id, option_id)
);

-- Direct Messaging
CREATE TABLE direct_messages (
    user_a_id UUID NOT NULL,
    user_b_id UUID NOT NULL,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_a_id, user_b_id),
    CONSTRAINT consistent_parties CHECK (user_a_id < user_b_id),
    FOREIGN KEY (user_a_id) REFERENCES users(id),
    FOREIGN KEY (user_b_id) REFERENCES users(id)
);

CREATE TABLE messages (
    sender_id UUID NOT NULL,
    recipient_id UUID NOT NULL,
    content TEXT NOT NULL,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (sender_id, recipient_id, creation_date)
);

-- Reports
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    thread_id BIGINT,
    comment_id BIGINT,
    reporter_id UUID NOT NULL,
    moderator_id UUID,
    report_reason TEXT NOT NULL,
    actions_taken TEXT,
    creation_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_date TIMESTAMPTZ,
    FOREIGN KEY (reporter_id) REFERENCES users(id),
    FOREIGN KEY (moderator_id) REFERENCES users(id),
    FOREIGN KEY (thread_id) REFERENCES threads(id),
    FOREIGN KEY (comment_id) REFERENCES comments(id),
    CHECK (thread_id IS NOT NULL OR comment_id IS NOT NULL)
);

-- Indexes the search vector for threads and comments

CREATE INDEX comments_search_vector_idx ON comments USING gin(search_vector);
CREATE INDEX threads_search_vector_idx ON threads USING gin(search_vector);

-- Trigger function to update threads table's search vector

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

-- Trigger function to update comments table's search vector

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

-- Trigger function to update user's comment count

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

-- Trigger function to update user's post count

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
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
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

-- Trigger functions to update threads' like and dislike count
-- whenever user_thread_likes table is updated / inserted into

CREATE OR REPLACE FUNCTION update_thread_like_count()
    RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP) = 'DELETE' THEN
        IF OLD.variant = 1 THEN
            UPDATE threads SET like_count = like_count - 1 WHERE id = OLD.thread_id;
        ELSIF OLD.variant = 0 THEN
            UPDATE threads SET dislike_count = dislike_count - 1 WHERE id = OLD.thread_id;
        END IF;
    ELSIF (TG_OP) = 'UPDATE' THEN
        -- When changing from like to dislike or vice versa
        IF NEW.variant = 1 AND OLD.variant = 0 THEN
            UPDATE threads SET like_count = like_count + 1, dislike_count = dislike_count - 1 WHERE id = NEW.thread_id;
        ELSIF NEW.variant = 0 AND OLD.variant = 1 THEN
            UPDATE threads SET like_count = like_count - 1, dislike_count = dislike_count + 1 WHERE id = NEW.thread_id;
        END IF;
    ELSIF (TG_OP) = 'INSERT' THEN
        IF NEW.variant = 1 THEN
            UPDATE threads SET like_count = like_count + 1 WHERE id = NEW.thread_id;
        ELSIF NEW.variant = 0 THEN
            UPDATE threads SET dislike_count = dislike_count + 1 WHERE id = NEW.thread_id;
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_thread_like_count_update
    AFTER INSERT OR DELETE OR UPDATE
    ON user_thread_likes
    FOR EACH ROW
EXECUTE FUNCTION update_thread_like_count();

-- Trigger functions to update comments' like and dislike count
-- whenever user_comment_likes table is updated / inserted into

CREATE OR REPLACE FUNCTION update_comment_like_count()
    RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP) = 'DELETE' THEN
        IF OLD.variant = 1 THEN
            UPDATE comments SET like_count = like_count - 1 WHERE id = OLD.comment_id;
        ELSIF OLD.variant = 0 THEN
            UPDATE comments SET dislike_count = dislike_count - 1 WHERE id = OLD.comment_id;
        END IF;
    ELSIF (TG_OP) = 'UPDATE' THEN
        -- When changing from like to dislike or vice versa
        IF NEW.variant = 1 AND OLD.variant = 0 THEN
            UPDATE comments SET like_count = like_count + 1, dislike_count = dislike_count - 1 WHERE id = NEW.comment_id;
        ELSIF NEW.variant = 0 AND OLD.variant = 1 THEN
            UPDATE comments SET like_count = like_count - 1, dislike_count = dislike_count + 1 WHERE id = NEW.comment_id;
        END IF;
    ELSIF (TG_OP) = 'INSERT' THEN
        IF NEW.variant = 1 THEN
            UPDATE comments SET like_count = like_count + 1 WHERE id = NEW.comment_id;
        ELSIF NEW.variant = 0 THEN
            UPDATE comments SET dislike_count = dislike_count + 1 WHERE id = NEW.comment_id;
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_comment_like_count_update
    AFTER INSERT OR DELETE OR UPDATE
    ON user_comment_likes
    FOR EACH ROW
EXECUTE FUNCTION update_comment_like_count();

-- Trigger function to update the last_comment_timestamp column every time
-- a new comment is added to a thread

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
    AFTER INSERT ON comments
    FOR EACH ROW
EXECUTE FUNCTION update_thread_last_comment_timestamp();

-- Trigger functions to update threads' comment count
-- whenever comments table is deleted from / inserted into

CREATE OR REPLACE FUNCTION update_thread_comment_count()
RETURNS TRIGGER AS $$
    BEGIN
        IF (TG_OP) = 'INSERT' THEN
            UPDATE threads
            SET comment_count = comment_count + 1
            WHERE threads.id = NEW.thread_id;
        ELSEIF (TG_OP) = 'DELETE' THEN
            UPDATE threads
            SET comment_count = comment_count - 1
            WHERE threads.id = OLD.thread_id;
        END IF;
        RETURN NULL;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_thread_comment_count_update
    AFTER INSERT OR DELETE
    ON comments
    FOR EACH ROW
EXECUTE FUNCTION update_thread_comment_count();

-- Tags and custom tags

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    tag TEXT NOT NULL
);

CREATE TABLE custom_tags (
    id BIGSERIAL PRIMARY KEY,
    tag TEXT NOT NULL
);

CREATE TABLE thread_custom_tags (
    thread_id BIGINT,
    custom_tag_id BIGINT,
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (custom_tag_id) REFERENCES custom_tags(id)
);

-- Indexes

CREATE UNIQUE INDEX idx_custom_tags_tag ON custom_tags(tag);
CREATE INDEX idx_thread_custom_tags ON thread_custom_tags(custom_tag_id);

-- Thread auxiliary tables

CREATE TABLE thread_tags (
    thread_id INT,
    tag_id INT,
    PRIMARY KEY (thread_id, tag_id),
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE thread_channels (
     thread_id INT,
     channel_id BIGINT,
     PRIMARY KEY (thread_id, channel_id),
     FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
     FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
);

-- Insert default tags

INSERT INTO tags (id, tag) VALUES
                               (0, 'Technology'),
                               (1, 'Gaming'),
                               (2, 'Entertainment'),
                               (3, 'Lifestyle'),
                               (4, 'Education'),
                               (5, 'Community'),
                               (6, 'Business'),
                               (7, 'Hobbies'),
                               (8, 'Science'),
                               (9, 'Sports'),
                               (10, 'Creative Arts'),
                               (11, 'Politics'),
                               (12, 'DIY & Crafting'),
                               (13, 'Automobiles'),
                               (14, 'Pets & Animals'),
                               (15, 'Health & Wellness'),
                               (16, 'Work & Productivity'),
                               (17, 'Travel'),
                               (18, 'Food & Drinks');
-- SAMPLE DATA
-- INSERT INTO users(id, username, password_hash) VALUES
--    ('23cdeffc-1c44-41e6-ab0b-001e6591b01f', 'kyo73', '$argon2id$v=19$m=65536,t=1,p=2$RdTX6X6yI9aNSDqsIEy5Aw$LA1cB0j7vDUzv21NQz8fvAvAtXRsdfHIioGKJ3e38Oo');
--
-- INSERT INTO user_profiles(id, creation_date) VALUES
--     ('23cdeffc-1c44-41e6-ab0b-001e6591b01f', '2025-01-15 13:23:12.799387+07');
--
-- INSERT INTO threads(user_id, title, original_post) VALUES
--     ('23cdeffc-1c44-41e6-ab0b-001e6591b01f', 'The Rust Programming Language', 'Rust is the best language. Rust is the best language. Rust is the best language. Rust is the best language. Rust is the best language. Rust is the best language. Rust is the best language.');
--
-- INSERT INTO thread_tags VALUES (1, 0);