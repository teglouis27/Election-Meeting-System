package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	//"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// App struct
type App struct {
	ctx         context.Context
	db          *sql.DB
	meetingDate time.Time
	echo        *echo.Echo
}

type CustomContext struct {
	echo.Context
	UserEmail string
	UserID    int
}

// Create a struct to receive survey data
type SurveyResponse struct {
	ResponseData struct {
		Vote struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"vote"`
		Nomination struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"nomination"`
		Feature struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"feature"`
		Spending struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"spending"`
		Question struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"question"`
		Election struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"election"`
		Threshold struct {
			QuestionType  string `json:"question_type"`
			QuestionText  string `json:"question_text"`
			ResponseValue string `json:"response_value"`
		} `json:"threshold"`
	} `json:"response_data"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		// You can set meeting date to any date you want
		meetingDate: time.Date(2025, time.February, 20, 2, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
		echo:        echo.New(),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) error {
	a.ctx = ctx
	if err := a.initDB(); err != nil {
		return err
	}

	// Setup Echo routes
	a.setupRoutes()
	// Start Echo server with error handling
	go func() {
		if err := a.echo.Start(":8080"); err != nil {
			// Log the error but don't crash the application
			println("Echo server error:", err.Error())
		}
	}()

	return nil
}

func (a *App) setupRoutes() {
	a.echo.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	a.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{
			echo.HeaderAuthorization,
			echo.HeaderContentType,
			echo.HeaderXRequestedWith,
		},
		ExposeHeaders:    []string{echo.HeaderContentLength},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	a.echo.POST("/login", a.handleLogin)
	a.echo.POST("/survey", a.handleSaveSurvey,
		echojwt.WithConfig(echojwt.Config{
			SigningKey:  []byte(os.Getenv("JWT_SECRET")),
			TokenLookup: "header:Authorization",
			ContextKey:  "user",
			NewClaimsFunc: func(c echo.Context) jwt.Claims {
				// Return an instance of your CustomClaims struct
				return &CustomClaims{}
			},
			ErrorHandler: func(c echo.Context, err error) error {
				log.Printf("JWT Error: %v", err)

				if errors.Is(err, echojwt.ErrJWTInvalid) {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Invalid token",
					})
				}

				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid request",
				})
			},
		}),
	)

	a.echo.GET("/election", a.handleElection)
}

func (a *App) handleLogin(c echo.Context) error {
	fmt.Println("Received login request")

	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&credentials); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	if credentials.Email == "" || credentials.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email and password are required",
		})
	}

	// Search user information
	var user struct {
		ID       int
		HasVoted bool
		Password string
	}

	err := a.db.QueryRow("SELECT password, has_voted FROM users WHERE email = ?",
		credentials.Email).Scan(&user.ID, &user.Password, &user.HasVoted)
	if err != nil {
		log.Println("Error:", err)

		if err == sql.ErrNoRows {
			log.Printf("User not found: %s", credentials.Email)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
		}
		log.Printf("Database error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error: " + err.Error()})
	}

	if user.Password != credentials.Password {
		log.Printf("Invalid password for user: %s", credentials.Email)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	// Create JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   credentials.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		log.Printf("JWT Signing Error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create token",
		})
	}

	// Configure redirectURL
	now := time.Now()
	isMeetingDay := now.Year() == a.meetingDate.Year() &&
		now.Month() == a.meetingDate.Month() &&
		now.Day() == a.meetingDate.Day()

	redirectURL := "/election"
	if isMeetingDay && !user.HasVoted {
		redirectURL = "/survey"
	}

	log.Printf("User %s logged in successfully", credentials.Email)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "Login successful",
		"token":       tokenString,
		"redirectURL": redirectURL,
		"user": map[string]interface{}{
			"email":    credentials.Email,
			"hasVoted": user.HasVoted,
		},
	})
}

type CustomClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (c *CustomContext) SetUser(email string, id int) {
	c.UserEmail = email
	c.UserID = id
}

// Middleware for configuring context
func UserContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := &CustomContext{Context: c}
		return next(cc)
	}
}

func (a *App) handleSaveSurvey(c echo.Context) error {
	log.Println("Received save-survey request")

	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(*CustomClaims)

	userID := claims.UserID
	userEmail := claims.Email

	log.Printf("Processing survey for user: %s (ID: %d)", userEmail, userID)

	// Check voting status
	var hasVoted bool
	err := a.db.QueryRowContext(
		context.Background(),
		"SELECT has_voted FROM users WHERE id = ?",
		userID,
	).Scan(&hasVoted)

	if err != nil {
		log.Printf("Database error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "An error occurred while checking the status",
		})
	}

	if hasVoted {
		log.Printf("User %s has already voted", userEmail)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "You have already voted",
		})
	}

	// Review and process survey data
	var survey SurveyResponse
	if err := c.Bind(&survey); err != nil {
		log.Printf("Invalid survey data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "The survey data format is invalid: " + err.Error(),
		})
	}

	// Validate survey data
	if err := validateSurvey(&survey); err != nil {
		log.Printf("Survey validation failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Survey data is invalid: " + err.Error(),
		})
	}

	// Convert to JSON
	voteData, err := json.Marshal(survey.ResponseData)
	if err != nil {
		log.Printf("JSON marshaling error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "An error occurred while processing the data",
		})
	}

	// Transaction management
	tx, err := a.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		log.Printf("Transaction start error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Unable to start data recording process",
		})
	}
	defer tx.Rollback()

	// Save vote from user
	if _, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO user_votes 
        (user_id, vote_choice, vote_timestamp)
        VALUES (?, ?, CURRENT_TIMESTAMP)`,
		userID,
		voteData,
	); err != nil {
		log.Printf("Vote insertion error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save vote",
		})
	}

	// Update user status
	if _, err = tx.ExecContext(
		context.Background(),
		"UPDATE users SET has_voted = TRUE WHERE id = ?",
		userID,
	); err != nil {
		log.Printf("User update error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "User status update failed",
		})
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Printf("Transaction commit error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Transaction commit is incomplete",
		})
	}

	log.Printf("Survey saved successfully for user: %s", userEmail)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "Survey saved",
		"redirectURL": "/election",
		"metadata": map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"userID":    userID,
		},
	})
}

func validateSurvey(survey *SurveyResponse) error {
	if survey.ResponseData.Vote.ResponseValue == "" {
		return fmt.Errorf("votes must be specified")
	}
	if !isValidVote(survey.ResponseData.Vote.ResponseValue) {
		return fmt.Errorf("invalid votes must be -1, 0, or 1")
	}
	if survey.ResponseData.Nomination.ResponseValue == "" {
		return fmt.Errorf("the name must be specified")
	}
	if len(survey.ResponseData.Nomination.ResponseValue) < 2 || len(survey.ResponseData.Nomination.ResponseValue) > 50 {
		return fmt.Errorf("์name must be between 2 and 50 characters long")
	}
	if survey.ResponseData.Feature.ResponseValue == "" {
		return fmt.Errorf("you must specify the features you want to add")
	}
	if survey.ResponseData.Spending.ResponseValue == "" {
		return fmt.Errorf("amount and purpose must be specified")
	}
	if !isValidSpending(survey.ResponseData.Spending.ResponseValue) {
		return fmt.Errorf("format for specifying the amount and purpose is invalid")
	}
	if survey.ResponseData.Question.ResponseValue == "" {
		return fmt.Errorf("question must be specified")
	}
	if survey.ResponseData.Election.ResponseValue == "" {
		return fmt.Errorf("time period for the next election must be specified")
	}
	weeks, err := parseElectionWeeks(survey.ResponseData.Election.ResponseValue)
	if err != nil {
		return fmt.Errorf("election period format is incorrect: %v", err)
	}
	if weeks < 1 || weeks > 24 {
		return fmt.Errorf("election period must be between 1 and 24 weeks")
	}
	if survey.ResponseData.Threshold.ResponseValue == "" {
		return fmt.Errorf("number of votes required for change must be specified")
	}
	if !isValidThreshold(survey.ResponseData.Threshold.ResponseValue) {
		return fmt.Errorf("number of votes is incorrect")
	}
	return nil
}

func isValidVote(vote string) bool {
	validVotes := []string{"-1", "0", "1"}
	for _, v := range validVotes {
		if vote == v {
			return true
		}
	}
	return false
}

func isValidSpending(spending string) bool {
	parts := strings.SplitN(spending, " for ", 2)
	if len(parts) != 2 {
		return false
	}
	_, err := strconv.ParseFloat(parts[0], 64)
	return err == nil && parts[1] != ""
}

func parseElectionWeeks(election string) (int, error) {
	parts := strings.Split(election, " ")
	if len(parts) != 2 || parts[1] != "weeks" {
		return 0, fmt.Errorf("format must be 'number of weeks'")
	}
	return strconv.Atoi(parts[0])
}

func isValidThreshold(threshold string) bool {
	num, err := strconv.Atoi(threshold)
	return err == nil && num > 0
}

func (a *App) handleElection(c echo.Context) error {
	// Handle election data
	return c.JSON(http.StatusOK, map[string]string{
		"data": "Election times and results go here",
	})
}

func (a *App) initDB() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	url := os.Getenv("DATABASE_URL")
	authToken := os.Getenv("AUTH_TOKEN")
	db, err := sql.Open("libsql", url+"?authToken="+authToken)
	if err != nil {
		return err
	}
	a.db = db

	_, err = a.db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return err
	}

	return nil
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
	if a.echo != nil {
		a.echo.Shutdown(ctx)
	}
}
