package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	firebase "firebase.google.com/go"
	fbAuth "firebase.google.com/go/auth"
	"google.golang.org/api/option"
)

// InitFirebaseAuth initializes a Firebase Admin SDK auth client using the
// GOOGLE_APPLICATION_CREDENTIALS environment variable for service account JSON.
// Returns nil if credentials are missing.
func InitFirebaseAuth(ctx context.Context) (*fbAuth.Client, error) {
	cred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if cred == "" {
		// Local dev convenience: if a Firebase Admin service account json exists in the
		// working directory, use it. This is intentionally best-effort.
		matches, _ := filepath.Glob("delivery-*-firebase-adminsdk-*.json")
		switch len(matches) {
		case 0:
			return nil, nil
		case 1:
			cred = matches[0]
		default:
			return nil, errors.New("multiple firebase service account json files found in working directory; set GOOGLE_APPLICATION_CREDENTIALS explicitly")
		}
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cred))
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}
