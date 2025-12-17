CREATE TABLE users (
	login VARCHAR(255) PRIMARY KEY NOT NULL,
	password VARCHAR(255) NOT NULL
);

INSERT INTO users (login, password)
VALUES ('admin', 'admin');
