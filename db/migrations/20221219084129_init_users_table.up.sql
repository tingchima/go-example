-- USERS
CREATE TABLE IF NOT EXISTS users(
    id serial NOT NULL,
    name VARCHAR (50) NOT NULL,
    PRIMARY KEY(id)
);