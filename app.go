package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
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

// สร้าง struct สำหรับรับข้อมูล survey
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
		meetingDate: time.Date(2025, time.February, 16, 2, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
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
	a.echo.Use(middleware.CORS())
	a.echo.Use(middleware.Recover())

	a.echo.POST("/login", a.handleLogin)
	a.echo.POST("/survey", a.handleSaveSurvey)
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

	var storedPassword string
	var hasVoted bool
	err := a.db.QueryRow("SELECT password, has_voted FROM users WHERE email = ?",
		credentials.Email).Scan(&storedPassword, &hasVoted)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error: " + err.Error()})
	}

	if storedPassword != credentials.Password {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	now := time.Now()
	isMeetingDay := now.Year() == a.meetingDate.Year() &&
		now.Month() == a.meetingDate.Month() &&
		now.Day() == a.meetingDate.Day()

	redirectURL := "/survey"
	if hasVoted {
		redirectURL = "/election"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Login successful",
		"isMeetingDay": isMeetingDay,
		"redirectURL":  redirectURL,
	})
}

func (a *App) handleSaveSurvey(c echo.Context) error {
	fmt.Println("Received save-survey request")
	// Get user email from the request (you might need to add this to the request)
	userEmail := c.Get("user_email").(string)

	var user struct {
		ID       int
		HasVoted bool
	}
	err := a.db.QueryRow("SELECT id, has_voted FROM users WHERE email = ?", userEmail).Scan(&user.ID, &user.HasVoted)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}

	if user.HasVoted {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "You have already voted"})
	}

	var survey SurveyResponse
	if err := c.Bind(&survey); err != nil {
		fmt.Printf("Error binding request: %v\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}
	fmt.Printf("Received survey data: %+v\n", survey)

	prettyJSON, err := json.MarshalIndent(survey.ResponseData, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling response data: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process survey data",
		})
	}
	fmt.Printf("Pretty Marshaled JSON:\n%s\n", string(prettyJSON))

	voteChoice, err := json.Marshal(survey.ResponseData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process survey data"})
	}

	tx, err := a.db.Begin()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Transaction error"})
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO user_votes (user_id, vote_choice, vote_timestamp)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, user.ID, voteChoice)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to record vote"})
	}

	_, err = tx.Exec("UPDATE users SET has_voted = TRUE WHERE id = ?", user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user status"})
	}

	if err = tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "Survey saved successfully",
		"redirectURL": "/election",
	})
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
