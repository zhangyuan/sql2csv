CREATE TABLE users AS
  SELECT * FROM (VALUES (1, 'Jack'), (2, 'James'), (3, 'Kitty')) AS t (id, name);
