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

type PromoTestSuite struct {
	suite.Suite
	router      *gin.Engine
	handler     *handlers.PromoHandler
	cfg         *config.Config
	adminToken  string
	userToken   string
	adminID     uint
	userID      uint
	promoCodeID uint
}

func (suite *PromoTestSuite) SetupSuite() {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	database.DB = db
	database.AutoMigrate()

	// Setup config
	suite.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-promo",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	// Create admin user
	admin := models.User{
		Name:     "Admin User",
		Email:    "admin@promo.com",
		Password: "admin123",
		Role:     "admin",
		IsActive: true,
	}
	database.DB.Create(&admin)
	suite.adminID = admin.ID

	// Create regular user
	user := models.User{
		Name:     "Regular User",
		Email:    "user@promo.com",
		Password: "user123",
		Role:     "user",
		IsActive: true,
	}
	database.DB.Create(&user)
	suite.userID = user.ID

	// Generate tokens
	adminToken, _ := middleware.GenerateToken(&admin, suite.cfg)
	suite.adminToken = adminToken

	userToken, _ := middleware.GenerateToken(&user, suite.cfg)
	suite.userToken = userToken

	// Create test promo code
	now := time.Now()
	promo := models.PromoCode{
		Code:            "TEST10",
		Description:     "Test 10% discount",
		Type:            "percentage",
		DiscountPercent: 10,
		MaxDiscount:     50000,
		MinPurchase:     0,
		MaxUses:         100,
		UsedCount:       0,
		MaxUsesPerUser:  1,
		IsActive:        true,
		StartsAt:        now,
		ExpiresAt:       now.AddDate(0, 1, 0),
		ApplicablePlans: "basic,premium",
	}
	database.DB.Create(&promo)
	suite.promoCodeID = promo.ID

	// Setup handler
	suite.handler = handlers.NewPromoHandler()

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	v1 := suite.router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(suite.cfg))
	{
		promos := v1.Group("/promos")
		{
			promos.GET("", suite.handler.GetPromoCodes)
			promos.POST("/validate", suite.handler.ValidatePromoCode)
			// promos.POST("/apply", suite.handler.ApplyPromoCode) // missing implementation
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.POST("/promos", suite.handler.CreatePromoCode)
			admin.PUT("/promos/:id", suite.handler.UpdatePromoCode)
			admin.DELETE("/promos/:id", suite.handler.DeletePromoCode)
			admin.GET("/promos/:id/usage", suite.handler.GetPromoUsageHistory)
		}
	}
}

func (suite *PromoTestSuite) TearDownSuite() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

func (suite *PromoTestSuite) SetupTest() {
	// Reset promo usage before each test
	database.DB.Exec("DELETE FROM promo_usages")
	database.DB.Model(&models.PromoCode{}).Where("id = ?", suite.promoCodeID).Update("used_count", 0)
}

func (suite *PromoTestSuite) TestGetPromoCodes() {
	req, _ := http.NewRequest("GET", "/api/v1/promos", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	promos := response["promo_codes"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(promos), 1)
}

func (suite *PromoTestSuite) TestValidatePromoCodeSuccess() {
	payload := map[string]interface{}{
		"code":   "TEST10",
		"amount": 100000,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/promos/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(suite.T(), true, response["valid"])
	assert.NotNil(suite.T(), response["discount_amount"])
}

func (suite *PromoTestSuite) TestValidatePromoCodeInvalid() {
	payload := map[string]interface{}{
		"code":   "INVALID",
		"amount": 100000,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/promos/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *PromoTestSuite) TestValidatePromoCodeExpired() {
	// Create expired promo
	expiredPromo := models.PromoCode{
		Code:            "EXPIRED",
		Description:     "Expired promo",
		Type:            "percentage",
		DiscountPercent: 20,
		IsActive:        true,
		StartsAt:        time.Now().AddDate(0, -2, 0),
		ExpiresAt:       time.Now().AddDate(0, -1, 0), // Expired
	}
	database.DB.Create(&expiredPromo)

	payload := map[string]interface{}{
		"code":   "EXPIRED",
		"amount": 100000,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/promos/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *PromoTestSuite) TestApplyPromoCode() {
	suite.T().Skip("ApplyPromoCode missing implementation")
}

func (suite *PromoTestSuite) TestCreatePromoCodeAsAdmin() {
	now := time.Now()
	payload := map[string]interface{}{
		"code":             "NEWPROMO",
		"description":      "New promo code",
		"type":             "percentage",
		"discount_percent": 15,
		"max_discount":     100000,
		"min_purchase":     50000,
		"max_uses":         50,
		"is_active":        true,
		"starts_at":        now.Format(time.RFC3339),
		"expires_at":       now.AddDate(0, 1, 0).Format(time.RFC3339),
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/admin/promos", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify promo was created
	var promo models.PromoCode
	err := database.DB.Where("code = ?", "NEWPROMO").First(&promo).Error
	assert.NoError(suite.T(), err)
}

func (suite *PromoTestSuite) TestCreatePromoCodeAsUser() {
	payload := map[string]interface{}{
		"code":        "USERPROMO",
		"description": "User trying to create promo",
		"type":        "percentage",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/admin/promos", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *PromoTestSuite) TestUpdatePromoCode() {
	payload := map[string]interface{}{
		"description":      "Updated description",
		"discount_percent": 20,
		"is_active":        false,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/promos/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify update
	var promo models.PromoCode
	database.DB.First(&promo, 1)
	assert.Equal(suite.T(), "Updated description", promo.Description)
	assert.Equal(suite.T(), false, promo.IsActive)
}

func (suite *PromoTestSuite) TestDeletePromoCode() {
	// Create promo to delete
	promoToDelete := models.PromoCode{
		Code:        "DELETEME",
		Description: "Delete this promo",
		Type:        "fixed",
		IsActive:    true,
		StartsAt:    time.Now(),
		ExpiresAt:   time.Now().AddDate(0, 1, 0),
	}
	database.DB.Create(&promoToDelete)

	req, _ := http.NewRequest("DELETE", "/api/v1/admin/promos/"+string(rune(promoToDelete.ID)), nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify soft delete
	var deletedPromo models.PromoCode
	err := database.DB.First(&deletedPromo, promoToDelete.ID).Error
	assert.Error(suite.T(), err)
}

func (suite *PromoTestSuite) TestGetPromoUsage() {
	// Create some usage records
	for i := 0; i < 5; i++ {
		usage := models.PromoUsage{
			PromoCodeID:    suite.promoCodeID,
			UserID:         suite.userID,
			OrderID:        1,
			DiscountAmount: 10000,
		}
		database.DB.Create(&usage)
	}

	req, _ := http.NewRequest("GET", "/api/v1/admin/promos/1/usage", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	usages := response["usages"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(usages), 5)
}

func TestPromoTestSuite(t *testing.T) {
	suite.Run(t, new(PromoTestSuite))
}
