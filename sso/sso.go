package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/lucalattore/goat"
	"gopkg.in/square/go-jose.v2/jwt"
)

// AuthParams describes the SSO parameters requered to validate authentication
type AuthParams struct {
	ValidationURL      string
	ValidationRequired bool
}

// AuthContextType is the type of auth context key identifier
type AuthContextType struct{}

// AuthContextKey identifies the profile in the auth context
var AuthContextKey = &AuthContextType{}

// AuthData is the type of auth data information saved into the context
type AuthData struct {
	Profile map[string]interface{}
	Token   string
	Claims  map[string]interface{}
}

// PreferredUsername get the preferred username
func (a AuthData) PreferredUsername() string {
	if username, ok := a.Profile["preferred_username"].(string); ok {
		return username
	}

	return ""
}

// AuthMiddleware checks the authentication
func (p AuthParams) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Request", r.Method, r.RequestURI)

		var token string
		splitToken := strings.Split(r.Header.Get("Authorization"), "Bearer ")
		if len(splitToken) > 1 {
			token = splitToken[1]
		}

		if token == "" {
			rr := goat.NewHTTPRequest(r)
			token = rr.Param("token")
		}

		authData := AuthData{Token: token}
		if token == "" {
			if p.ValidationRequired {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		} else {
			log.Println("Found token:", token)

			// decode JWT token without verifying the signature
			tk, err := jwt.ParseSigned(token)
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			err = tk.UnsafeClaimsWithoutVerification(&authData.Claims)
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			log.Println("Got JWT:", authData.Claims, err)
			if p.ValidationURL != "" {
				clientID, ok := authData.Claims["client_id"].(string)
				if !ok {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				profile, err := validateToken(p.ValidationURL, clientID, token)
				if err != nil {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				authData.Profile = *profile
			}
		}

		c := context.WithValue(r.Context(), AuthContextKey, authData)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r.WithContext(c))
	})
}

func validateToken(url string, clientID string, token string) (*map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url+"?client_id="+clientID, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("SSO Validator returned:", err)
		return nil, err
	}

	defer res.Body.Close()

	fmt.Printf("Client Proto: %d\n", res.ProtoMajor)
	if res.StatusCode != 200 {
		return nil, errors.New("Authorization failed")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var profile map[string]interface{}
	err = json.Unmarshal(body, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}
