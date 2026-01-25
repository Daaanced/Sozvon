// Authorization\handlers\register_user_service.go
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type userProfileRequest struct {
	Login   string `json:"login"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Info    string `json:"info"`
	Picture string `json:"picture"`
}

func createUserProfile(login string) error {
	body := userProfileRequest{
		Login:   login,
		Name:    login,
		Email:   "",
		Info:    "",
		Picture: "",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		"http://localhost:8083/users",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.New("user service returned non-201 status")
	}

	return nil
}
