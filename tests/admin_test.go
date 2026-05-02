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

type AdminTestSuite struct {
	suite.Suite
	router           *gin.Engine
	handler          *handlers.AdminHandler
	cfg              *config.Config
	adminToken       string
	userToken        string
	adminID          uint
	regularUserID    uint
}

func (suite *AdminTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	database.DB = db
	database.AutoMigrate()

	// Setup config
	suite.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-admin",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	// Create admin user
	admin := models.User{
		Name:             "Admin User",
		Email:            "admin@test.com",
		Password:         "admin123",
		Role:             "admin",
		IsActive:         true,
		SubscriptionPlan: "premium",
	}
	database.DB.Create(&admin)
	suite.adminID = admin.ID

	// Create regular user
	user := models.User{
		Name:             "Regular User",
		Email:            "user@test.com",
		Password:         "user123",
		Role:             "user",
		IsActive:         true,
		SubscriptionPlan: "free",
	}
	database.DB.Create(&user)
	suite.regularUserID = user.ID

	// Generate tokens
	adminToken, _ := middleware.GenerateToken(&admin, suite.cfg)
	suite.adminToken = adminToken

	userToken, _ := middleware.GenerateToken(&user, suite.cfg)
	suite.userToken = userToken

	// Setup handler
	suite.handler = handlers.NewAdminHandler()

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	v1 := suite.router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(suite.cfg))
	{
		admin := v1.Group("/admin")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/users", suite.handler.GetAllUsers)
			admin.GET("/users/:id", suite.handler.GetUser)
			admin.PUT("/users/:id/status", suite.handler.UpdateUserStatus)
			admin.DELETE("/users/:id", suite.handler.DeleteUser)
			admin.GET("/stats", suite.handler.GetDashboardStats)
			admin.GET("/revenue", suite.handler.GetRevenueReport)
			admin.GET("/templates/analytics", suite.handler.GetTemplateAnalytics)
			admin.GET("/growth", suite.handler.GetUserGrowth)
		}
	}
}

func (suite *AdminTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *AdminTestSuite) TestGetUsersAsAdmin() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	users := response["users"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(users), 2)
}

func (suite *AdminTestSuite) TestGetUsersAsRegularUser() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *AdminTestSuite) TestGetUsersPagination() {
	// Create more users
	for i := 0; i < 15; i++ {
		user := models.User{
			Name:     "Test User",
			Email:    "test" + string(rune(i)) + "@test.com",
			Password: "password123",
			IsActive: true,
		}
		database.DB.Create(&user)
	}

	req, _ := http.NewRequest("GET", "/api/v1/admin/users?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	users := response["users"].([]interface{})
	assert.Equal(suite.T(), 10, len(users))
}

func (suite *AdminTestSuite) TestGetSingleUser() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/users/2", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	user := response["user"].(map[string]interface{})
	assert.NotNil(suite.T(), user)
}

func (suite *AdminTestSuite) TestUpdateUser() {
	payload := map[string]interface{}{
		"is_active": false,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/users/2/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify update
	var user models.User
	database.DB.First(&user, 2)
	assert.Equal(suite.T(), false, user.IsActive)
}

func (suite *AdminTestSuite) TestDeleteUser() {
	// Create user to delete
	userToDelete := models.User{
		Name:     "Delete Me",
		Email:    "delete@test.com",
		Password: "password123",
		IsActive: true,
	}
	database.DB.Create(&userToDelete)

	req, _ := http.NewRequest("DELETE", "/api/v1/admin/users/"+string(rune(userToDelete.ID)), nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify user is soft deleted
	var deletedUser models.User
	err := database.DB.First(&deletedUser, userToDelete.ID).Error
	assert.Error(suite.T(), err)
}

func (suite *AdminTestSuite) TestGetStats() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotNil(suite.T(), response["total_users"])
	assert.NotNil(suite.T(), response["total_photos"])
	assert.NotNil(suite.T(), response["total_revenue"])
}

func (suite *AdminTestSuite) TestGetAnalytics() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/revenue?period=month", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotNil(suite.T(), response["report"])
}

func (suite *AdminTestSuite) TestAdminEndpointsWithoutAuth() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
