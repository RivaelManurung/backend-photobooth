package tests

import (
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TemplateTestSuite struct {
	suite.Suite
	router  *gin.Engine
	handler *handlers.TemplateHandler
}

func (suite *TemplateTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)
	
	database.DB = db
	database.AutoMigrate()

	// Setup handler
	storageService := &services.StorageService{}
	suite.handler = handlers.NewTemplateHandler(storageService)

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	
	templates := suite.router.Group("/templates")
	{
		templates.GET("", suite.handler.GetTemplates)
		templates.GET("/:id", suite.handler.GetTemplate)
		templates.GET("/categories", suite.handler.GetTemplateCategories)
	}
}

func (suite *TemplateTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *TemplateTestSuite) SetupTest() {
	// Clean and seed templates
	database.DB.Exec("DELETE FROM templates")
	
	templates := []models.Template{
		{
			Name:            "Classic White",
			Slug:            "classic-white",
			Category:        "classic",
			BackgroundColor: "#ffffff",
			IsActive:        true,
			IsPremium:       false,
		},
		{
			Name:            "Premium Gold",
			Slug:            "premium-gold",
			Category:        "premium",
			BackgroundColor: "#ffd700",
			IsActive:        true,
			IsPremium:       true,
		},
	}
	
	for _, t := range templates {
		database.DB.Create(&t)
	}
}

func (suite *TemplateTestSuite) TestGetTemplates() {
	req, _ := http.NewRequest("GET", "/templates", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	templates := response["templates"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(templates), 1)
}

func (suite *TemplateTestSuite) TestGetTemplatesByCategory() {
	req, _ := http.NewRequest("GET", "/templates?category=classic", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	templates := response["templates"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(templates), 1)
}

func (suite *TemplateTestSuite) TestGetSingleTemplate() {
	// Get first template
	var template models.Template
	database.DB.First(&template)

	req, _ := http.NewRequest("GET", "/templates/"+fmt.Sprint(template.ID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *TemplateTestSuite) TestGetTemplateCategories() {
	req, _ := http.NewRequest("GET", "/templates/categories", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	categories := response["categories"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(categories), 1)
}

func TestTemplateTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateTestSuite))
}
