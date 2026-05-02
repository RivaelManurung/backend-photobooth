package tests

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type AuthTestSuite struct {
	suite.Suite
	router  *gin.Engine
	handler *handlers.AuthHandler
	cfg     *config.Config
}

func (suite *AuthTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)
	
	database.DB = db
	database.AutoMigrate()

	// Setup config
	suite.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}

	// Setup handler
	suite.handler = handlers.NewAuthHandler(suite.cfg)

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	
	auth := suite.router.Group("/auth")
	{
		auth.POST("/register", suite.handler.Register)
		auth.POST("/login", suite.handler.Login)
	}
}

func (suite *AuthTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *AuthTestSuite) SetupTest() {
	// Clean database before each test
	database.DB.Exec("DELETE FROM users")
}

func (suite *AuthTestSuite) TestRegisterSuccess() {
	payload := map[string]interface{}{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotEmpty(suite.T(), response["access_token"])
	assert.NotEmpty(suite.T(), response["refresh_token"])
	assert.NotNil(suite.T(), response["user"])
}

func (suite *AuthTestSuite) TestRegisterDuplicateEmail() {
	// Create first user - password will be hashed by BeforeCreate hook
	user := models.User{
		Name:     "Existing User",
		Email:    "existing@example.com",
		Password: "password123",
		IsActive: true,
	}
	database.DB.Create(&user)

	// Try to register with same email
	payload := map[string]interface{}{
		"name":     "New User",
		"email":    "existing@example.com",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(suite.T(), response["error"], "already registered")
}

func (suite *AuthTestSuite) TestRegisterInvalidEmail() {
	payload := map[string]interface{}{
		"name":     "Test User",
		"email":    "invalid-email",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *AuthTestSuite) TestLoginSuccess() {
	// Create user - password will be hashed by BeforeCreate hook
	user := models.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
		IsActive: true,
	}
	database.DB.Create(&user)

	// Login
	payload := map[string]interface{}{
		"email":    "test@example.com",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotEmpty(suite.T(), response["access_token"])
	assert.NotEmpty(suite.T(), response["refresh_token"])
}

func (suite *AuthTestSuite) TestLoginInvalidCredentials() {
	payload := map[string]interface{}{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *AuthTestSuite) TestLoginWrongPassword() {
	// Create user - password will be hashed by BeforeCreate hook
	user := models.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "correctpassword",
		IsActive: true,
	}
	database.DB.Create(&user)

	// Try wrong password
	payload := map[string]interface{}{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
