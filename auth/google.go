package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"example/hello/db"
)

var (
	googleOauthConfig *oauth2.Config
	configOnce        sync.Once
)

func getGoogleOauthConfig() *oauth2.Config {
	configOnce.Do(func() {
		googleOauthConfig = &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		}
	})
	return googleOauthConfig
}

const oauthStateString = "chess-app-state"

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func GoogleLogin(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, getGoogleOauthConfig().AuthCodeURL(oauthStateString))
}

func GoogleCallback(c *gin.Context) {
	ctx := context.Background()

	if c.Query("state") != oauthStateString {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	token, err := getGoogleOauthConfig().Exchange(ctx, c.Query("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "exchange failed"})
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch userinfo"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read userinfo"})
		return
	}

	var gUser googleUserInfo
	if err := json.Unmarshal(body, &gUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse userinfo"})
		return
	}

	user, err := db.UpsertUserByGoogleID(ctx, gUser.ID, gUser.Email, gUser.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert user"})
		return
	}

	sessionToken, err := CreateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"?token="+sessionToken)
}
