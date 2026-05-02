package tests

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"backendphotobooth/services"
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

type PhotoTestSuite struct {
	suite.Suite
	router      *gin.Engine
	handler     *handlers.PhotoHandler
	cfg         *config.Config
	accessToken string
	userID      uint
	templateID  uint
	sessionID   string
}

func (suite *PhotoTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	database.DB = db
	database.AutoMigrate()

	// Setup config
	suite.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-photo",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	// Create test user
	user := models.User{
		Name:             "Photo Test User",
		Email:            "photo@test.com",
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

	// Create test session
	session := models.Session{
		UserID:     &user.ID,
		SessionID:  "test-session-001",
		TemplateID: template.ID,
		Status:     "active",
	}
	database.DB.Create(&session)
	suite.sessionID = session.SessionID

	// Setup handler
	storageService := &services.StorageService{}
	imageProcessor := &services.ImageProcessor{}
	suite.handler = handlers.NewPhotoHandler(storageService, imageProcessor)

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	v1 := suite.router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(suite.cfg))
	{
		photos := v1.Group("/photos")
		{
			photos.GET("", suite.handler.GetPhotos)
			photos.GET("/:id", suite.handler.GetPhoto)
			photos.DELETE("/:id", suite.handler.DeletePhoto)
		}
	}
}

func (suite *PhotoTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *PhotoTestSuite) SetupTest() {
	// Clean photos before each test
	database.DB.Exec("DELETE FROM photos")
}

func (suite *PhotoTestSuite) TestGetPhotosEmpty() {
	req, _ := http.NewRequest("GET", "/api/v1/photos", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	photos := response["photos"].([]interface{})
	assert.Equal(suite.T(), 0, len(photos))
}

func (suite *PhotoTestSuite) TestGetPhotosWithData() {
	// Create test photo
	photo := models.Photo{
		UserID:     suite.userID,
		SessionID:  suite.sessionID,
		TemplateID: suite.templateID,
		Status:     "completed",
		StoragePath: "/test/photo.jpg",
	}
	database.DB.Create(&photo)

	req, _ := http.NewRequest("GET", "/api/v1/photos", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	photos := response["photos"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(photos), 1)
}

func (suite *PhotoTestSuite) TestGetPhotosPagination() {
	// Create multiple photos
	for i := 0; i < 15; i++ {
		photo := models.Photo{
			UserID:     suite.userID,
			SessionID:  suite.sessionID,
			TemplateID: suite.templateID,
			Status:     "completed",
			StoragePath:   "/test/photo.jpg",
		}
		database.DB.Create(&photo)
	}

	req, _ := http.NewRequest("GET", "/api/v1/photos?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	photos := response["photos"].([]interface{})
	assert.Equal(suite.T(), 10, len(photos))
	assert.Equal(suite.T(), float64(15), response["total"])
}

func (suite *PhotoTestSuite) TestGetSinglePhoto() {
	// Create test photo
	photo := models.Photo{
		UserID:     suite.userID,
		SessionID:  suite.sessionID,
		TemplateID: suite.templateID,
		Status:     "completed",
		StoragePath: "/test/photo.jpg",
	}
	database.DB.Create(&photo)

	req, _ := http.NewRequest("GET", "/api/v1/photos/1", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *PhotoTestSuite) TestGetPhotoNotFound() {
	req, _ := http.NewRequest("GET", "/api/v1/photos/999", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *PhotoTestSuite) TestDeletePhoto() {
	// Create test photo
	photo := models.Photo{
		UserID:     suite.userID,
		SessionID:  suite.sessionID,
		TemplateID: suite.templateID,
		Status:     "completed",
		StoragePath: "/test/photo.jpg",
	}
	database.DB.Create(&photo)

	req, _ := http.NewRequest("DELETE", "/api/v1/photos/1", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify photo is deleted
	var deletedPhoto models.Photo
	err := database.DB.First(&deletedPhoto, photo.ID).Error
	assert.Error(suite.T(), err) // Should be soft deleted
}

func (suite *PhotoTestSuite) TestDeletePhotoUnauthorized() {
	// Create photo for another user
	otherUser := models.User{
		Name:     "Other User",
		Email:    "other@test.com",
		Password: "password123",
		IsActive: true,
	}
	database.DB.Create(&otherUser)

	photo := models.Photo{
		UserID:     otherUser.ID,
		SessionID:  suite.sessionID,
		TemplateID: suite.templateID,
		Status:     "completed",
		StoragePath: "/test/photo.jpg",
	}
	database.DB.Create(&photo)

	req, _ := http.NewRequest("DELETE", "/api/v1/photos/1", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *PhotoTestSuite) TestGetPhotosWithoutAuth() {
	req, _ := http.NewRequest("GET", "/api/v1/photos", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func TestPhotoTestSuite(t *testing.T) {
	suite.Run(t, new(PhotoTestSuite))
}
