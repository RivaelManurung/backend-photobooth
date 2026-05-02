package tests

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/middleware"
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

type IntegrationTestSuite struct {
	suite.Suite
	router      *gin.Engine
	cfg         *config.Config
	accessToken string
	userID      uint
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)
	
	database.DB = db
	database.AutoMigrate()

	// Setup config
	suite.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-for-integration-testing",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	// Setup router with all handlers
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	
	authHandler := handlers.NewAuthHandler(suite.cfg)
	
	v1 := suite.router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}
		
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(suite.cfg))
		{
			profile := protected.Group("/profile")
			{
				profile.GET("", authHandler.GetProfile)
				profile.PUT("", authHandler.UpdateProfile)
			}
		}
	}
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *IntegrationTestSuite) TestCompleteUserFlow() {
	// Step 1: Register
	registerPayload := map[string]interface{}{
		"name":     "Integration Test User",
		"email":    "integration@example.com",
		"password": "password123",
	}

	body, _ := json.Marshal(registerPayload)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var registerResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &registerResponse)

	// Debug: print response if registration failed
	if w.Code != http.StatusCreated {
		suite.T().Logf("Registration failed with status %d: %v", w.Code, registerResponse)
	}

	// Check if registration was successful
	if registerResponse["access_token"] == nil {
		suite.T().Fatalf("Registration failed: %v", registerResponse)
		return
	}

	suite.accessToken = registerResponse["access_token"].(string)
	user := registerResponse["user"].(map[string]interface{})
	suite.userID = uint(user["id"].(float64))

	assert.NotEmpty(suite.T(), suite.accessToken)

	// Step 2: Get Profile
	req, _ = http.NewRequest("GET", "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Debug: print response if profile fetch failed
	if w.Code != http.StatusOK {
		suite.T().Logf("Get profile failed with status %d, token: %s, body: %s", w.Code, suite.accessToken, w.Body.String())
	}

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var profileResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &profileResponse)

	profileUser := profileResponse["user"].(map[string]interface{})
	assert.Equal(suite.T(), "Integration Test User", profileUser["name"])

	// Step 3: Update Profile
	updatePayload := map[string]interface{}{
		"name":  "Updated Integration User",
		"phone": "+6281234567890",
	}

	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PUT", "/api/v1/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Step 4: Verify Update
	req, _ = http.NewRequest("GET", "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &profileResponse)
	updatedUser := profileResponse["user"].(map[string]interface{})
	assert.Equal(suite.T(), "Updated Integration User", updatedUser["name"])
	assert.Equal(suite.T(), "+6281234567890", updatedUser["phone"])

	// Step 5: Logout and try to access protected route
	req, _ = http.NewRequest("GET", "/api/v1/profile", nil)
	// No authorization header

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *IntegrationTestSuite) TestInvalidTokenAccess() {
	req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
