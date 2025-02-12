-- This sample data is generated by ChatGPT and then personally vetted for appropriateness.

-- Connect to the database
\c forum

-- Insert 10 users
INSERT INTO users (id, username, password_hash)
SELECT gen_random_uuid(),
       CONCAT('user', i),
       CONCAT('hashed_password_', i)
FROM generate_series(1, 10) i;

-- Insert 10 user profiles
INSERT INTO user_profiles (id, profile_picture_path, biodata, email, post_count, comment_count, preferred_lang, preferred_theme)
SELECT id,
       CONCAT('profile_pics/user', i, '.png'),
       CASE
           WHEN i = 1 THEN 'Hi, I love discussing anime and manga!'
           WHEN i = 2 THEN 'Tech enthusiast and traveler.'
           WHEN i = 3 THEN 'Fitness is my passion.'
           WHEN i = 4 THEN 'Exploring the role of AI in our lives.'
           WHEN i = 5 THEN 'Manga and gaming fan.'
           WHEN i = 6 THEN 'Sci-fi movies are my thing.'
           WHEN i = 7 THEN 'Lover of Japanese snacks.'
           WHEN i = 8 THEN 'Team player in multiplayer games.'
           WHEN i = 9 THEN 'New to exploring Tokyo!'
           ELSE 'Motivation is key to success.'
           END,
       CONCAT('user', i, '@example.com'),
       FLOOR(RANDOM() * 20),
       FLOOR(RANDOM() * 50),
       'en',
       'auto'
FROM users u
         JOIN generate_series(1, 10) i ON u.username = CONCAT('user', i);

-- Insert 10 threads
INSERT INTO threads (id, user_id, title, original_post, like_count, dislike_count, view_count)
SELECT i, u.id,
       CASE
           WHEN i = 1 THEN 'Best Anime of 2023'
           WHEN i = 2 THEN 'Tips for Visiting Tokyo'
           WHEN i = 3 THEN 'Hottest Gaming Gear This Year'
           WHEN i = 4 THEN 'Exploring Manga Classics'
           WHEN i = 5 THEN 'Must-Try Japanese Snacks'
           WHEN i = 6 THEN 'How to Stay Motivated for Fitness'
           WHEN i = 7 THEN 'The Role of AI in Modern Life'
           WHEN i = 8 THEN 'Travel Tips for First-Time Japan Visitors'
           WHEN i = 9 THEN 'Underrated Sci-Fi Movies to Watch'
           ELSE 'Best Strategies for Online Multiplayer Games'
           END,
       CASE
           WHEN i = 1 THEN 'Let’s discuss the anime that captured our hearts this year.'
           WHEN i = 2 THEN 'From transportation to food, here’s what you need to know.'
           WHEN i = 3 THEN 'What gaming gear has impressed you the most recently?'
           WHEN i = 4 THEN 'Manga enthusiasts, let’s talk about the classics.'
           WHEN i = 5 THEN 'Snacks like Pocky and KitKat are iconic. Share your favorites!'
           WHEN i = 6 THEN 'Staying fit is tough. Share your strategies here!'
           WHEN i = 7 THEN 'AI is changing everything. Let’s discuss its impact.'
           WHEN i = 8 THEN 'Are you planning your first trip to Japan? Here’s how to prepare.'
           WHEN i = 9 THEN 'Share and discuss lesser-known gems in sci-fi.'
           ELSE 'Teamwork and communication are key. Share your tips!'
           END,
       FLOOR(RANDOM() * 30),
       FLOOR(RANDOM() * 5),
       FLOOR(RANDOM() * 100)
FROM users u
         JOIN generate_series(1, 10) i ON u.username = CONCAT('user', i);

-- Insert 10 tags
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

-- Insert 10 comments for each thread
-- INSERT INTO comments (thread_id, user_id, comment, like_count, dislike_count)
-- SELECT t.id,
--        u.id,
--        CASE
--            WHEN t.title = 'Best Anime of 2023' THEN 'Demon Slayer was amazing this year!'
--            WHEN t.title = 'Tips for Visiting Tokyo' THEN 'Tokyo Tower is a must-visit.'
--            WHEN t.title = 'Hottest Gaming Gear This Year' THEN 'The new Corsair keyboard is awesome.'
--            WHEN t.title = 'Exploring Manga Classics' THEN 'One Piece has been incredible lately.'
--            WHEN t.title = 'Must-Try Japanese Snacks' THEN 'Matcha KitKats are my favorite.'
--            WHEN t.title = 'How to Stay Motivated for Fitness' THEN 'Staying consistent is key.'
--            WHEN t.title = 'The Role of AI in Modern Life' THEN 'AI will change how we work and live.'
--            WHEN t.title = 'Travel Tips for First-Time Japan Visitors' THEN 'Get a JR Pass to save on train travel.'
--            WHEN t.title = 'Underrated Sci-Fi Movies to Watch' THEN '"Ex Machina" is brilliant.'
--            ELSE 'Communication is the key to winning.'
--            END,
--        FLOOR(RANDOM() * 20),
--        FLOOR(RANDOM() * 5)
-- FROM threads t
--          JOIN users u ON u.username = CONCAT('user', ((t.id - 1) % 10) + 1)
-- ORDER BY t.id, u.username;
