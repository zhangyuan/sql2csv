CREATE TABLE users AS
SELECT *
FROM (
    VALUES ROW(1, 'Jack'),
      ROW(2, 'James'),
      ROW(3, cast(NULL as VARCHAR(10))),
      ROW(4, 'y\'c, z')
  ) t (id, name);
