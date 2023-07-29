CREATE TABLE users AS
SELECT *
FROM (
    VALUES (1, 'Jack'),
      (2, 'James'),
      (3, 'Kitty'),
      (4, 'y''c, z')
  ) AS t (id, name);
