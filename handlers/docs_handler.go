package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type DocsHandler struct{}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// GetSwaggerUI serves Swagger UI
func (h *DocsHandler) GetSwaggerUI(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Photo Booth API - Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/api/v1/docs/swagger.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                docExpansion: "list",
                filter: true,
                tryItOutEnabled: true
            });
            window.ui = ui;
        };
    </script>
</body>
</html>
`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// GetSwaggerJSON returns Swagger JSON spec
func (h *DocsHandler) GetSwaggerJSON(c *gin.Context) {
	c.File("./docs/swagger.json")
}

// GetAPIDocs returns API documentation
func (h *DocsHandler) GetAPIDocs(c *gin.Context) {
	docs := gin.H{
		"title":       "Photo Booth API Documentation",
		"version":     "1.0.0",
		"description": "Complete API documentation for Photo Booth Backend",
		"base_url":    "http://localhost:8080/api/v1",
		"endpoints": map[string]interface{}{
			"authentication": []map[string]interface{}{
				{
					"method":      "POST",
					"path":        "/auth/register",
					"description": "Register new user",
					"body": map[string]string{
						"name":     "string (required)",
						"email":    "string (required, email format)",
						"password": "string (required, min 6 chars)",
						"phone":    "string (optional)",
					},
					"response": map[string]string{
						"access_token":  "JWT token",
						"refresh_token": "Refresh token",
						"user":          "User object",
					},
				},
				{
					"method":      "POST",
					"path":        "/auth/login",
					"description": "Login user",
					"body": map[string]string{
						"email":    "string (required)",
						"password": "string (required)",
					},
					"response": map[string]string{
						"access_token":  "JWT token",
						"refresh_token": "Refresh token",
						"user":          "User object",
					},
				},
				{
					"method":      "POST",
					"path":        "/auth/refresh",
					"description": "Refresh access token",
					"body": map[string]string{
						"refresh_token": "string (required)",
					},
				},
				{
					"method":      "POST",
					"path":        "/auth/logout",
					"description": "Logout user",
					"auth":        "Bearer token required",
				},
			},
			"profile": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/profile",
					"description": "Get current user profile",
					"auth":        "Bearer token required",
				},
				{
					"method":      "PUT",
					"path":        "/profile",
					"description": "Update user profile",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"name":   "string",
						"phone":  "string",
						"avatar": "string (URL)",
					},
				},
				{
					"method":      "PUT",
					"path":        "/profile/password",
					"description": "Change password",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"old_password": "string (required)",
						"new_password": "string (required)",
					},
				},
			},
			"templates": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/templates",
					"description": "Get all templates",
					"query": map[string]string{
						"category":  "string (optional)",
						"is_active": "boolean (optional)",
						"search":    "string (optional)",
					},
				},
				{
					"method":      "GET",
					"path":        "/templates/:id",
					"description": "Get single template",
				},
				{
					"method":      "GET",
					"path":        "/templates/categories",
					"description": "Get all template categories",
				},
				{
					"method":      "POST",
					"path":        "/templates",
					"description": "Create new template (Admin only)",
					"auth":        "Bearer token required (Admin)",
				},
			},
			"photos": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/photos",
					"description": "Get user photos",
					"auth":        "Bearer token required",
				},
				{
					"method":      "POST",
					"path":        "/photos",
					"description": "Upload photo",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"template_id": "uint (required)",
						"session_id":  "uint (optional)",
						"file":        "multipart/form-data",
					},
				},
				{
					"method":      "GET",
					"path":        "/photos/:id",
					"description": "Get single photo",
					"auth":        "Bearer token required",
				},
				{
					"method":      "DELETE",
					"path":        "/photos/:id",
					"description": "Delete photo",
					"auth":        "Bearer token required",
				},
			},
			"payments_gopay": []map[string]interface{}{
				{
					"method":      "POST",
					"path":        "/payments/gopay/qris",
					"description": "Create GoPay QRIS payment",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"order_id": "uint (required)",
						"amount":   "float64 (required)",
					},
					"response": map[string]string{
						"qr_code":    "Base64 QR code image",
						"qr_string":  "QRIS string",
						"payment_id": "Payment ID",
						"expires_at": "Expiration time",
					},
				},
				{
					"method":      "GET",
					"path":        "/payments/gopay/:id/status",
					"description": "Check payment status",
					"auth":        "Bearer token required",
				},
				{
					"method":      "POST",
					"path":        "/webhooks/gopay",
					"description": "GoPay webhook callback",
					"note":        "Called by GoPay server",
				},
			},
			"orders": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/orders",
					"description": "Get user orders",
					"auth":        "Bearer token required",
				},
				{
					"method":      "POST",
					"path":        "/orders",
					"description": "Create new order",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"items":      "array of order items",
						"promo_code": "string (optional)",
					},
				},
				{
					"method":      "GET",
					"path":        "/orders/:id",
					"description": "Get single order",
					"auth":        "Bearer token required",
				},
			},
			"sessions": []map[string]interface{}{
				{
					"method":      "POST",
					"path":        "/sessions",
					"description": "Create photo session",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"template_id": "uint (required)",
						"name":        "string (optional)",
					},
				},
				{
					"method":      "GET",
					"path":        "/sessions",
					"description": "Get user sessions",
					"auth":        "Bearer token required",
				},
				{
					"method":      "GET",
					"path":        "/sessions/:id",
					"description": "Get single session with photos",
					"auth":        "Bearer token required",
				},
			},
			"promo_codes": []map[string]interface{}{
				{
					"method":      "POST",
					"path":        "/promo/validate",
					"description": "Validate promo code",
					"auth":        "Bearer token required",
					"body": map[string]string{
						"code": "string (required)",
					},
				},
			},
			"admin": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/admin/dashboard",
					"description": "Get dashboard statistics",
					"auth":        "Bearer token required (Admin)",
				},
				{
					"method":      "GET",
					"path":        "/admin/users",
					"description": "Get all users with pagination",
					"auth":        "Bearer token required (Admin)",
					"query": map[string]string{
						"page":   "int",
						"limit":  "int",
						"search": "string",
						"plan":   "string",
					},
				},
				{
					"method":      "GET",
					"path":        "/admin/users/:id",
					"description": "Get single user details",
					"auth":        "Bearer token required (Admin)",
				},
				{
					"method":      "PUT",
					"path":        "/admin/users/:id/status",
					"description": "Update user status",
					"auth":        "Bearer token required (Admin)",
				},
				{
					"method":      "GET",
					"path":        "/admin/revenue",
					"description": "Get revenue report",
					"auth":        "Bearer token required (Admin)",
					"query": map[string]string{
						"period": "day|week|month|year",
					},
				},
			},
			"websocket": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/ws",
					"description": "WebSocket connection for real-time notifications",
					"protocol":    "WebSocket",
				},
			},
			"health": []map[string]interface{}{
				{
					"method":      "GET",
					"path":        "/health",
					"description": "Health check endpoint",
				},
			},
		},
		"authentication": map[string]string{
			"type":   "Bearer Token",
			"header": "Authorization: Bearer <token>",
			"note":   "Get token from /auth/login or /auth/register",
		},
		"default_credentials": map[string]string{
			"admin_email":    "admin@photobooth.com",
			"admin_password": "admin123",
		},
		"promo_codes": []map[string]string{
			{
				"code":        "WELCOME10",
				"discount":    "10%",
				"description": "Welcome discount for all users",
			},
			{
				"code":        "FIRST50",
				"discount":    "50%",
				"description": "First time user discount",
			},
			{
				"code":        "YEARLY20",
				"discount":    "20%",
				"description": "Yearly subscription discount",
			},
		},
	}

	c.JSON(http.StatusOK, docs)
}

// GetAPIDocsHTML returns HTML documentation
func (h *DocsHandler) GetAPIDocsHTML(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Photo Booth API Documentation</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6;
            color: #333;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 20px;
            text-align: center;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        h1 { font-size: 2.5em; margin-bottom: 10px; }
        .subtitle { font-size: 1.2em; opacity: 0.9; }
        .info-box {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .info-box h2 {
            color: #667eea;
            margin-bottom: 15px;
            font-size: 1.5em;
        }
        .endpoint-group {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .endpoint-group h3 {
            color: #667eea;
            margin-bottom: 15px;
            text-transform: capitalize;
            font-size: 1.3em;
        }
        .endpoint {
            border-left: 4px solid #667eea;
            padding: 15px;
            margin-bottom: 15px;
            background: #f9f9f9;
            border-radius: 4px;
        }
        .method {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-weight: bold;
            font-size: 0.85em;
            margin-right: 10px;
        }
        .method.get { background: #61affe; color: white; }
        .method.post { background: #49cc90; color: white; }
        .method.put { background: #fca130; color: white; }
        .method.delete { background: #f93e3e; color: white; }
        .path {
            font-family: 'Courier New', monospace;
            font-size: 1.1em;
            color: #333;
            font-weight: 600;
        }
        .description {
            margin-top: 8px;
            color: #666;
        }
        .auth-badge {
            display: inline-block;
            background: #ff6b6b;
            color: white;
            padding: 2px 8px;
            border-radius: 3px;
            font-size: 0.75em;
            margin-left: 10px;
        }
        .code-block {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 15px;
            border-radius: 4px;
            overflow-x: auto;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            margin-top: 10px;
        }
        .promo-code {
            display: inline-block;
            background: #667eea;
            color: white;
            padding: 5px 15px;
            border-radius: 20px;
            margin: 5px;
            font-weight: bold;
        }
        .base-url {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            margin: 10px 0;
        }
        .credentials {
            background: #fff3cd;
            border: 1px solid #ffc107;
            padding: 15px;
            border-radius: 4px;
            margin: 15px 0;
        }
        .credentials strong { color: #856404; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>📸 Photo Booth API</h1>
            <p class="subtitle">Complete REST API Documentation v1.0.0</p>
        </header>

        <div class="info-box">
            <h2>🚀 Base URL</h2>
            <div class="base-url">http://localhost:8080/api/v1</div>
        </div>

        <div class="info-box">
            <h2>🔐 Authentication</h2>
            <p>Most endpoints require Bearer token authentication. Include the token in the Authorization header:</p>
            <div class="code-block">Authorization: Bearer &lt;your_token&gt;</div>
            <p style="margin-top: 10px;">Get your token from <code>/auth/login</code> or <code>/auth/register</code> endpoints.</p>
        </div>

        <div class="info-box">
            <h2>👤 Default Admin Credentials</h2>
            <div class="credentials">
                <strong>Email:</strong> admin@photobooth.com<br>
                <strong>Password:</strong> admin123
            </div>
        </div>

        <div class="info-box">
            <h2>🎟️ Available Promo Codes</h2>
            <span class="promo-code">WELCOME10 (10% off)</span>
            <span class="promo-code">FIRST50 (50% off)</span>
            <span class="promo-code">YEARLY20 (20% off)</span>
        </div>

        <div class="endpoint-group">
            <h3>🔑 Authentication</h3>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/auth/register</span>
                <p class="description">Register a new user account</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/auth/login</span>
                <p class="description">Login with email and password</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/auth/refresh</span>
                <p class="description">Refresh access token</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/auth/logout</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Logout current user</p>
            </div>
        </div>

        <div class="endpoint-group">
            <h3>👤 Profile</h3>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/profile</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Get current user profile</p>
            </div>
            <div class="endpoint">
                <span class="method put">PUT</span>
                <span class="path">/profile</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Update user profile</p>
            </div>
            <div class="endpoint">
                <span class="method put">PUT</span>
                <span class="path">/profile/password</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Change password</p>
            </div>
        </div>

        <div class="endpoint-group">
            <h3>🎨 Templates</h3>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/templates</span>
                <p class="description">Get all photo templates</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/templates/:id</span>
                <p class="description">Get single template by ID</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/templates/categories</span>
                <p class="description">Get all template categories</p>
            </div>
        </div>

        <div class="endpoint-group">
            <h3>📸 Photos</h3>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/photos</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Get user's photos</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/photos</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Upload new photo</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/photos/:id</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Get single photo</p>
            </div>
            <div class="endpoint">
                <span class="method delete">DELETE</span>
                <span class="path">/photos/:id</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Delete photo</p>
            </div>
        </div>

        <div class="endpoint-group">
            <h3>💳 GoPay QRIS Payments</h3>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/payments/gopay/qris</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Create QRIS payment (returns QR code)</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/payments/gopay/:id/status</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Check payment status</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/webhooks/gopay</span>
                <p class="description">GoPay webhook callback (called by GoPay)</p>
            </div>
        </div>

        <div class="endpoint-group">
            <h3>🛒 Orders</h3>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/orders</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Get user's orders</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/orders</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Create new order</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/orders/:id</span>
                <span class="auth-badge">🔒 Auth Required</span>
                <p class="description">Get single order details</p>
            </div>
        </div>

        <div class="endpoint-group">
            <h3>⚙️ Admin (Admin Only)</h3>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/admin/dashboard</span>
                <span class="auth-badge">🔒 Admin Only</span>
                <p class="description">Get dashboard statistics</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/admin/users</span>
                <span class="auth-badge">🔒 Admin Only</span>
                <p class="description">Get all users with pagination</p>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/admin/revenue</span>
                <span class="auth-badge">🔒 Admin Only</span>
                <p class="description">Get revenue reports</p>
            </div>
        </div>

        <div class="info-box">
            <h2>📡 WebSocket</h2>
            <p>Connect to WebSocket for real-time notifications:</p>
            <div class="code-block">ws://localhost:8080/api/v1/ws</div>
            <p style="margin-top: 10px;">Receive real-time updates for:</p>
            <ul style="margin-left: 20px; margin-top: 10px;">
                <li>Photo processing completion</li>
                <li>Payment status updates</li>
                <li>Order confirmations</li>
                <li>System notifications</li>
            </ul>
        </div>

        <div class="info-box">
            <h2>🏥 Health Check</h2>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/health</span>
                <p class="description">Check API health status</p>
            </div>
        </div>

        <div class="info-box">
            <h2>📚 Full JSON Documentation</h2>
            <p>Get complete API documentation in JSON format:</p>
            <div class="code-block">GET /api/v1/docs/json</div>
        </div>

        <footer style="text-align: center; padding: 40px 0; color: #999;">
            <p>Photo Booth API v1.0.0 | Built with ❤️ using Go & Gin</p>
        </footer>
    </div>
</body>
</html>
`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
