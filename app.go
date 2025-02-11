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
	InstanceID   int `json:"instance_id"`
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
		meetingDate: time.Date(2025, time.February, 12, 2, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
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
	a.echo.POST("/login", a.handleLogin)
	a.echo.POST("/survey", a.handleSurvey)
	a.echo.GET("/election", a.handleElection)
	a.echo.POST("/save-survey", a.handleSaveSurvey)
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
	err := a.db.QueryRow("SELECT password FROM users WHERE email = ?", credentials.Email).Scan(&storedPassword)
	if err != nil || storedPassword != credentials.Password {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	now := time.Now()
	isMeetingDay := now.Year() == a.meetingDate.Year() &&
		now.Month() == a.meetingDate.Month() &&
		now.Day() == a.meetingDate.Day()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Login successful",
		"isMeetingDay": isMeetingDay,
		"data":         nil,
	})
}

func (a *App) handleSaveSurvey(c echo.Context) error {
	var survey SurveyResponse
	if err := c.Bind(&survey); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// แปลง response_data เป็น JSON string
	responseDataJSON, err := json.Marshal(survey.ResponseData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process survey data",
		})
	}

	// บันทึกลงฐานข้อมูล
	query := `INSERT INTO survey_responses (instance_id, response_data) VALUES (?, ?)`
	result, err := a.db.Exec(query, survey.InstanceID, string(responseDataJSON))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save survey: " + err.Error(),
		})
	}

	id, _ := result.LastInsertId()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Survey saved successfully",
		"id":      id,
	})
}

func (a *App) handleSurvey(c echo.Context) error {
	// Handle survey data
	return c.JSON(http.StatusOK, map[string]string{
		"data": "Survey data goes here",
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
