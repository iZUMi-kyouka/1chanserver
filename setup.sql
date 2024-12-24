DROP DATABASE IF EXISTS forum WITH (FORCE);

CREATE DATABASE forum;

\c forum 

CREATE TYPE app_language AS ENUM ('en', 'id', 'ja');
CREATE TYPE app_theme AS ENUM ('light', 'dark', 'auto');

CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE user_profiles (
    id UUID PRIMARY KEY,
    profile_picture_path TEXT,
    biodata TEXT DEFAULT 'どうも、1chanを使いますよ！',
    email TEXT,
    post_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    preferred_lang app_language DEFAULT 'en',
    preferred_theme app_theme DEFAULT 'auto',
    creation_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    FOREIGN KEY (id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID,
    token_hash TEXT NOT NULL,
    expiration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE threads (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    title TEXT NOT NULL,
    original_post TEXT NOT NULL,
    creation_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    like_count INT NOT NULL,
    dislike_count INT NOT NULL,
    view_count INT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    tag TEXT NOT NULL
);

CREATE TABLE thread_tags (
    thread_id BIGINT,
    tag_id INT,
    PRIMARY KEY (thread_id, tag_id),
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE channels (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT 'このチャンネルへようこそ！',
    profile_picture_path TEXT
);

CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    thread_id BIGINT NOT NULL,
    user_id UUID NOT NULL,
    comment TEXT NOT NULL,
    creation_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    like_count INT NOT NULL,
    dislike_count INT NOT NULL,
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);