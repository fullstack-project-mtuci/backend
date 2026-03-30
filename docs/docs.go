package docs

import (
	"encoding/json"
	"strings"

	"github.com/swaggo/swag"
)

// SwaggerInfo describes base swagger metadata.
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/api/v1",
	Schemes:          []string{},
	Title:            "Travel Portal API",
	Description:      "REST API for the Business Trip and Expense Portal",
	InfoInstanceName: "swagger",
}

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "description": "{{.Description}}",
        "title": "{{.Title}}",
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "schemes": {{marshal .Schemes}},
    "paths": {
        "/auth/register": {"post": {"tags": ["Auth"], "summary": "Register employee", "responses": {"201": {"description": "Created"}}}},
        "/auth/login": {"post": {"tags": ["Auth"], "summary": "Login with email/password", "responses": {"200": {"description": "OK"}}}},
        "/auth/refresh": {"post": {"tags": ["Auth"], "summary": "Refresh JWT", "responses": {"200": {"description": "OK"}}}},
        "/auth/me": {"get": {"tags": ["Auth"], "summary": "Current user profile", "responses": {"200": {"description": "OK"}}}},

        "/trip-requests": {
            "get": {"tags": ["Trips"], "summary": "List trip requests", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Trips"], "summary": "Create trip request", "responses": {"201": {"description": "Created"}}}
        },
        "/trip-requests/{id}": {
            "get": {"tags": ["Trips"], "summary": "Get trip request", "responses": {"200": {"description": "OK"}}},
            "put": {"tags": ["Trips"], "summary": "Update trip request", "responses": {"200": {"description": "OK"}}},
            "delete": {"tags": ["Trips"], "summary": "Delete trip request", "responses": {"204": {"description": "No Content"}}}
        },
        "/trip-requests/{id}/status": {"patch": {"tags": ["Trips"], "summary": "Change trip status", "responses": {"200": {"description": "OK"}}}},

        "/trip-requests/{tripId}/advance": {
            "get": {"tags": ["Advances"], "summary": "Get advance", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Advances"], "summary": "Create or update draft advance", "responses": {"201": {"description": "Created"}}}
        },
        "/trip-requests/{tripId}/advance/status": {"patch": {"tags": ["Advances"], "summary": "Change advance status", "responses": {"200": {"description": "OK"}}}},

        "/trip-requests/{tripId}/expense-report": {
            "get": {"tags": ["ExpenseReports"], "summary": "Get report for trip", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["ExpenseReports"], "summary": "Create expense report for trip", "responses": {"201": {"description": "Created"}}}
        },

        "/expense-reports/{reportId}": {"get": {"tags": ["ExpenseReports"], "summary": "Get report by id", "responses": {"200": {"description": "OK"}}}},
        "/expense-reports/{reportId}/items": {"post": {"tags": ["ExpenseReports"], "summary": "Add expense item", "responses": {"201": {"description": "Created"}}}},
        "/expense-reports/{reportId}/items/{itemId}": {
            "put": {"tags": ["ExpenseReports"], "summary": "Update expense item", "responses": {"200": {"description": "OK"}}},
            "delete": {"tags": ["ExpenseReports"], "summary": "Delete expense item", "responses": {"204": {"description": "No Content"}}}
        },
        "/expense-reports/{reportId}/status": {"patch": {"tags": ["ExpenseReports"], "summary": "Change report status", "responses": {"200": {"description": "OK"}}}},

        "/receipts": {
            "get": {"tags": ["Receipts"], "summary": "List uploaded receipts", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Receipts"], "summary": "Upload receipt and run OCR", "responses": {"201": {"description": "Created"}}}
        },

        "/admin/users": {
            "get": {"tags": ["Admin"], "summary": "List users", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Admin"], "summary": "Create user", "responses": {"201": {"description": "Created"}}}
        },
        "/admin/users/{id}": {"put": {"tags": ["Admin"], "summary": "Update user", "responses": {"200": {"description": "OK"}}}},

        "/admin/departments": {
            "get": {"tags": ["Admin"], "summary": "List departments", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Admin"], "summary": "Create department", "responses": {"201": {"description": "Created"}}}
        },
        "/admin/departments/{id}": {
            "put": {"tags": ["Admin"], "summary": "Update department", "responses": {"200": {"description": "OK"}}},
            "delete": {"tags": ["Admin"], "summary": "Delete department", "responses": {"204": {"description": "No Content"}}}
        },

        "/admin/projects": {
            "get": {"tags": ["Admin"], "summary": "List projects", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Admin"], "summary": "Create project", "responses": {"201": {"description": "Created"}}}
        },
        "/admin/projects/{id}": {
            "put": {"tags": ["Admin"], "summary": "Update project", "responses": {"200": {"description": "OK"}}},
            "delete": {"tags": ["Admin"], "summary": "Delete project", "responses": {"204": {"description": "No Content"}}}
        },

        "/admin/categories": {
            "get": {"tags": ["Admin"], "summary": "List expense categories", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Admin"], "summary": "Create category", "responses": {"201": {"description": "Created"}}}
        },
        "/admin/categories/{id}": {
            "put": {"tags": ["Admin"], "summary": "Update category", "responses": {"200": {"description": "OK"}}},
            "delete": {"tags": ["Admin"], "summary": "Delete category", "responses": {"204": {"description": "No Content"}}}
        },

        "/admin/budgets": {
            "get": {"tags": ["Admin"], "summary": "List budgets", "responses": {"200": {"description": "OK"}}},
            "post": {"tags": ["Admin"], "summary": "Create budget", "responses": {"201": {"description": "Created"}}}
        },

        "/audit/{entityType}/{entityId}/approvals": {"get": {"tags": ["Audit"], "summary": "List approval actions", "responses": {"200": {"description": "OK"}}}},
        "/audit/{entityType}/{entityId}/logs": {"get": {"tags": ["Audit"], "summary": "List audit log entries", "responses": {"200": {"description": "OK"}}}}
    }
}`

type swaggerDoc struct{}

func (d *swaggerDoc) ReadDoc() string {
	doc := docTemplate
	replacements := map[string]string{
		"{{.Description}}": SwaggerInfo.Description,
		"{{.Title}}":       SwaggerInfo.Title,
		"{{.Version}}":     SwaggerInfo.Version,
		"{{.Host}}":        SwaggerInfo.Host,
		"{{.BasePath}}":    SwaggerInfo.BasePath,
	}
	for placeholder, value := range replacements {
		doc = strings.ReplaceAll(doc, placeholder, value)
	}
	doc = strings.Replace(doc, "{{marshal .Schemes}}", marshalSchemes(SwaggerInfo.Schemes), 1)
	return doc
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), &swaggerDoc{})
}

func marshalSchemes(schemes []string) string {
	data, err := json.Marshal(schemes)
	if err != nil {
		return "[]"
	}
	return string(data)
}
