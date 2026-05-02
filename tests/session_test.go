package tests

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SessionTestSuite struct {
	suite.Suite
	router      *gin.Engine
	handler     *handlers.SessionHandler
	cfg         *config.Config
	accessToken string
	userID      uint
	templateID  uint
	sessionID   uint
}

func (suite *SessionTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	database.DB = db
	database.AutoMigrate()

	// Setup config
	suite.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-session",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	// Create test user
	user := models.User{
		Name:             "Session Test User",
		Email:            "session@test.com",
		Password:         "password123",
		IsActive:         true,
		SubscriptionPlan: "premium",
	}
	database.DB.Create(&user)
	suite.userID = user.ID

	// Generate token
	token, _ := middleware.GenerateToken(&user, suite.cfg)
	suite.accessToken = token

	// Create test template
	template := models.Template{
		Name:         "Test Template",
		Slug:         "test-template",
		Category:     "classic",
		LayoutType:   "single",
		PhotoCount:   1,
		Orientation:  "portrait",
		IsActive:     true,
		IsPremium:    false,
		RequiredPlan: "free",
	}
	database.DB.Create(&template)
	suite.templateID = template.ID

	// Setup handler
	suite.handler = handlers.NewSessionHandler()

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	v1 := suite.router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(suite.cfg))
	{
		sessions := v1.Group("/sessions")
		{
			sessions.POST("", suite.handler.CreateSession)
			sessions.GET("", suite.handler.GetUserSessions)
			sessions.GET("/:id", suite.handler.GetSession)
			sessions.PUT("/:id", suite.handler.UpdateSession)
			sessions.DELETE("/:id", suite.handler.DeleteSession)
			sessions.POST("/:id/complete", suite.handler.EndSession)
		}
	}
}

func (suite *SessionTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *SessionTestSuite) SetupTest() {
	// Clean sessions before each test
	database.DB.Exec("DELETE FROM sessions")
}

func (suite *SessionTestSuite) TestCreateSession() {
	payload := map[string]interface{}{
		"template_id": suite.templateID,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/sessions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	session := response["session"].(map[string]interface{})
	assert.NotNil(suite.T(), session["id"])
	assert.Equal(suite.T(), "active", session["status"])
}

func (suite *SessionTestSuite) TestCreateSessionInvalidTemplate() {
	payload := map[string]interface{}{
		"template_id": 999, // Non-existent template
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/sessions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *SessionTestSuite) TestGetSessions() {
	// Create test sessions
	for i := 0; i < 3; i++ {
		session := models.Session{
			UserID:     &suite.userID,
			TemplateID: suite.templateID,
			Status:     "active",
		}
		database.DB.Create(&session)
	}

	req, _ := http.NewRequest("GET", "/api/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	sessions := response["sessions"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(sessions), 3)
}

func (suite *SessionTestSuite) TestGetSessionsPagination() {
	// Create multiple sessions
	for i := 0; i < 15; i++ {
		session := models.Session{
			UserID:     &suite.userID,
			TemplateID: suite.templateID,
			Status:     "active",
		}
		database.DB.Create(&session)
	}

	req, _ := http.NewRequest("GET", "/api/v1/sessions?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	sessions := response["sessions"].([]interface{})
	assert.Equal(suite.T(), 10, len(sessions))
	assert.Equal(suite.T(), float64(15), response["total"])
}

func (suite *SessionTestSuite) TestGetSingleSession() {
	// Create test session
	session := models.Session{
		UserID:     &suite.userID,
		TemplateID: suite.templateID,
		Status:     "active",
	}
	database.DB.Create(&session)

	req, _ := http.NewRequest("GET", "/api/v1/sessions/1", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	sessionData := response["session"].(map[string]interface{})
	assert.Equal(suite.T(), "active", sessionData["status"])
}

func (suite *SessionTestSuite) TestGetSessionNotFound() {
	req, _ := http.NewRequest("GET", "/api/v1/sessions/999", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *SessionTestSuite) TestUpdateSession() {
	// Create test session
	session := models.Session{
		UserID:     &suite.userID,
		TemplateID: suite.templateID,
		Status:     "active",
	}
	database.DB.Create(&session)

	payload := map[string]interface{}{
		"status": "paused",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/api/v1/sessions/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify update
	var updatedSession models.Session
	database.DB.First(&updatedSession, 1)
	assert.Equal(suite.T(), "paused", updatedSession.Status)
}

func (suite *SessionTestSuite) TestCompleteSession() {
	// Create test session
	session := models.Session{
		UserID:     &suite.userID,
		TemplateID: suite.templateID,
		Status:     "active",
	}
	database.DB.Create(&session)

	req, _ := http.NewRequest("POST", "/api/v1/sessions/1/complete", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify session is completed
	var completedSession models.Session
	database.DB.First(&completedSession, 1)
	assert.Equal(suite.T(), "completed", completedSession.Status)
}

func (suite *SessionTestSuite) TestDeleteSession() {
	// Create test session
	session := models.Session{
		UserID:     &suite.userID,
		TemplateID: suite.templateID,
		Status:     "active",
	}
	database.DB.Create(&session)

	req, _ := http.NewRequest("DELETE", "/api/v1/sessions/1", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify session is soft deleted
	var deletedSession models.Session
	err := database.DB.First(&deletedSession, 1).Error
	assert.Error(suite.T(), err)
}

func (suite *SessionTestSuite) TestDeleteSessionUnauthorized() {
	// Create session for another user
	otherUser := models.User{
		Name:     "Other User",
		Email:    "other@session.com",
		Password: "password123",
		IsActive: true,
	}
	database.DB.Create(&otherUser)

	session := models.Session{
		UserID:     &otherUser.ID,
		TemplateID: suite.templateID,
		Status:     "active",
	}
	database.DB.Create(&session)

	req, _ := http.NewRequest("DELETE", "/api/v1/sessions/1", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *SessionTestSuite) TestSessionEndpointsWithoutAuth() {
	req, _ := http.NewRequest("GET", "/api/v1/sessions", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func TestSessionTestSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
