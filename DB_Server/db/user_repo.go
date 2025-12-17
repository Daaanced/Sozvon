package db

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func GetAllUsers() ([]User, error) {
	rows, err := DB.Query("SELECT login, password FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var u User
		err := rows.Scan(&u.Login, &u.Password)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func CreateUser(login, password string) error {
	_, err := DB.Exec(
		"INSERT INTO users (login, password) VALUES ($1, $2)",
		login, password,
	)
	return err
}
