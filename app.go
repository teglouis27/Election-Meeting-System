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

	//"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// App struct
type App struct {
	ctx  context.Context
	db   *sql.DB
	echo *echo.Echo
}

type CustomContext struct {
	echo.Context
	UserID int
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
		echo: echo.New(),
	}
}

func (a *App) beginTx(ctx context.Context) (*sql.Tx, error) {
	return a.db.BeginTx(ctx, nil)
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) error {
	a.ctx = ctx
	if err := a.initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
		return err
	}

	// Setup Echo routes
	a.setupRoutes()

	// Task Scheduler for reset vote status when meeting end
	go func() {
		if err := a.resetVotingStatus(); err != nil {
			log.Printf("Initial reset error: %v", err)
		}
		ticker := time.NewTicker(55 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := a.resetVotingStatus(); err != nil {
					log.Printf("Error resetting voting status: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

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
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	a.echo.Use(middleware.Logger())
	a.echo.Use(middleware.Recover())

	a.echo.POST("/login", a.handleLogin)

	authenticated := a.echo.Group("")
	authenticated.Use(UserContextMiddleware)
	authenticated.Use(a.AuthMiddleware)
	authenticated.POST("/survey", a.handleSaveSurvey)
	authenticated.GET("/election", a.handleElection)

	// Check authentication status
	a.echo.GET("/check-auth", func(c echo.Context) error {
		userID := c.Request().Header.Get("X-User-ID")
		return c.JSON(http.StatusOK, map[string]bool{"authenticated": userID != ""})
	})
}

func (a *App) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Request().Header.Get("X-User-ID")
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
		}

		var exists bool
		err := a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid user"})
		}

		return next(c)
	}
}

func (a *App) handleLogin(c echo.Context) error {
	fmt.Println("Received login request")

	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&credentials); err != nil {
		log.Printf("Invalid request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	log.Printf("Login attempt with email: %s", credentials.Email)

	if credentials.Email == "" || credentials.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email and password are required",
		})
	}

	// Search user information
	var user struct {
		ID       int
		Password string
		HasVoted bool
	}

	err := a.db.QueryRow("SELECT id, password, has_voted FROM users WHERE email = ?",
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

	log.Printf("Login successful for user ID: %d", user.ID)

	sessionStart, sessionEnd, err := a.getLatestVotingSession()
	if err != nil {
		log.Printf("Warning: %v, using default date", err)
		sessionStart = time.Time{}
		sessionEnd = time.Time{}
	}

	now := time.Now()
	isMeetingDay := !sessionStart.IsZero() && !sessionEnd.IsZero() &&
		now.After(sessionStart) && now.Before(sessionEnd)

	redirectURL := "/election"
	if isMeetingDay && !user.HasVoted {
		redirectURL = "/survey"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "Login successful",
		"redirectURL": redirectURL,
		"user": map[string]interface{}{
			"email":    credentials.Email,
			"hasVoted": user.HasVoted,
			"id":       user.ID,
		},
	})
}

func (c *CustomContext) SetUser(email string, id int) {
	c.UserID = id
}

// Middleware for configuring context
func UserContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userIDStr := c.Request().Header.Get("X-User-ID")
		log.Printf("X-User-ID Header received: %s", userIDStr)

		if userIDStr == "" {
			log.Printf("Empty User ID received")
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "User ID is required",
			})
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil || userID <= 0 {
			log.Printf("Invalid User ID: '%s', Error: %v", userIDStr, err)
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid User ID",
			})
		}

		log.Printf("Valid User ID received: %d", userID)

		cc := &CustomContext{
			Context: c,
			UserID:  userID,
		}
		return next(cc)
	}
}

func (a *App) getLatestVotingSession() (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	err := a.db.QueryRowContext(
		context.Background(),
		"SELECT start_date, end_date FROM voting_sessions ORDER BY start_date DESC LIMIT 1",
	).Scan(&startDate, &endDate)

	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, time.Time{}, fmt.Errorf("no voting sessions found")
		}
		return time.Time{}, time.Time{}, err
	}
	return startDate, endDate, nil
}

func (a *App) getCurrentVotingSessionID() (int, error) {
	var sessionID int
	err := a.db.QueryRowContext(
		context.Background(),
		`SELECT id FROM voting_sessions 
         WHERE start_date <= CURRENT_TIMESTAMP AND end_date >= CURRENT_TIMESTAMP 
         ORDER BY start_date DESC LIMIT 1`,
	).Scan(&sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no active voting session found")
		}
		return 0, err
	}
	return sessionID, nil
}

func (a *App) handleSaveSurvey(c echo.Context) error {
	log.Println("Received save-survey request")

	cc, ok := c.(*CustomContext)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Context configuration error",
		})
	}

	if cc.UserID <= 0 {
		log.Printf("Invalid User ID: %d", cc.UserID)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not authenticated",
		})
	}

	var user struct {
		HasVoted bool
	}

	err := a.db.QueryRowContext(
		context.Background(),
		"SELECT has_voted FROM users WHERE id = ?",
		cc.UserID, // ใช้ UserID จาก Custom Context
	).Scan(&user.HasVoted)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("User %d not found", cc.UserID)
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
		log.Printf("Database error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Database error",
		})
	}
	log.Printf("User %d has voted: %v", cc.UserID, user.HasVoted)
	if user.HasVoted {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "User has already voted"})
	}

	// Validate survey data
	var survey SurveyResponse
	if err := c.Bind(&survey); err != nil {
		log.Printf("Invalid survey data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "The survey data format is invalid: " + err.Error(),
		})
	}

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

	// Verify voting_session_id
	log.Println("Verifying current voting session")
	votingSessionID, err := a.getCurrentVotingSessionID()
	if err != nil {
		log.Printf("Error getting current voting session: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "No active voting session",
		})
	}
	log.Printf("Current voting session ID: %d", votingSessionID)

	// Transaction management
	log.Println("Starting database transaction")
	tx, err := a.beginTx(context.Background())
	if err != nil {
		log.Printf("Transaction start error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Unable to start data recording process",
		})
	}
	defer tx.Rollback()

	// Save vote from user
	log.Printf("Attempting to insert vote for user %d with session %d", cc.UserID, votingSessionID)
	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO user_votes
        (user_id, voting_session_id, vote_choice, vote_timestamp)
        VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		cc.UserID,
		votingSessionID,
		voteData,
	)
	if err != nil {
		log.Printf("Vote insertion error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to record vote",
		})
	}
	log.Printf("Vote successfully inserted for user %d", cc.UserID)

	// Update user status
	log.Printf("Updating has_voted status for user %d", cc.UserID)
	_, err = tx.ExecContext(
		context.Background(),
		"UPDATE users SET has_voted = TRUE WHERE id = ?",
		cc.UserID,
	)
	if err != nil {
		log.Printf("User update error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "User status update failed",
		})
	}
	log.Printf("User %d status updated successfully", cc.UserID)

	// Commit transaction
	log.Println("Committing transaction")
	if err = tx.Commit(); err != nil {
		log.Printf("Transaction commit error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Transaction commit is incomplete",
		})
	}
	log.Println("Transaction committed successfully")

	log.Printf("Survey saved successfully for user: %d", cc.UserID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "Survey saved",
		"redirectURL": "/election",
		"metadata": map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"userID":    cc.UserID,
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
	if len(survey.ResponseData.Nomination.ResponseValue) < 1 || len(survey.ResponseData.Nomination.ResponseValue) > 99 {
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
	parts := strings.Split(spending, " for ")
	if len(parts) != 2 {
		return false
	}
	amount, err := strconv.ParseFloat(parts[0], 64)
	return err == nil && amount > 0
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
	return c.File("election-times-and-results.html")
}

func (a *App) resetVotingStatus() error {
	_, err := a.db.Exec(`
        UPDATE users SET has_voted = 0 
        WHERE has_voted = 1 AND id IN (
            SELECT user_id FROM user_votes 
            WHERE voting_session_id = (
                SELECT id FROM voting_sessions 
                WHERE end_date < CURRENT_TIMESTAMP
                ORDER BY end_date DESC LIMIT 1
            )
        )
    `)
	return err
}

func (a *App) initDB() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	url := os.Getenv("DATABASE_URL")
	authToken := os.Getenv("AUTH_TOKEN")
	db, err := sql.Open("libsql", url+"?authToken="+authToken)
	if err != nil {
		return err
	}
	a.db = db
	a.db.SetMaxOpenConns(25)
	a.db.SetMaxIdleConns(5)
	a.db.SetConnMaxLifetime(30 * time.Minute)

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
