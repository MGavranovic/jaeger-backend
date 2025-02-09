package urlparser

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

/*
NOTE:
   ParseURL to get the path and the basePath from the func it's called in and return errors which are to be logged in the function it's being called from
   it also takes in the w ResponseWriter so we can send the errors with parsing as http

   I will have to customize this function to be able to parse for User IDs for example and potentialy other things needed from the URL
*/

func ParseURL(path, basePath string, w http.ResponseWriter) (string, error) {
	if !strings.HasPrefix(path, basePath) {
		log.Printf("Invalid URL: %s", path)
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return "", fmt.Errorf("Invalid URL: %s", path)
	}

	email := strings.TrimPrefix(path, basePath) // extracing the email from the path
	if email == "" {
		log.Printf("Email missing in the url request: %s", email)
		http.Error(w, "Email is required", http.StatusBadRequest)
		return "", fmt.Errorf("Email missing in the url request: %s", email)
	}
	return email, nil
}
