package email

import (
	"html/template"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestEmailManager_getMailBody(t *testing.T) {
	mockFS := fstest.MapFS{
		"templates/valid.html": &fstest.MapFile{
			Data: []byte(`<html><body>Hello, {{.Name}}!</body></html>`),
			Mode: 0755,
		},
		"templates/invalid.html": &fstest.MapFile{
			Data: []byte(`<html><body>Hello, {{.Name!}</body></html>`), // Invalid template syntax
			Mode: 0755,
		},
		"templates/error.html": &fstest.MapFile{
			Data: []byte(`<html><body>{{ if eq .Value "error" }}{{ .NonExistentField.Something }}{{ end }}</body></html>`),
			Mode: 0755,
		},
	}

	tests := []struct {
		name           string
		filePath       string
		templateName   string
		mailBuilder    interface{}
		cacheTemplates bool
		wantErr        bool
		expectedBody   string
		expectedErrMsg string
	}{
		{
			name:           "Valid template with data",
			filePath:       "templates/valid.html",
			templateName:   "valid.html",
			mailBuilder:    map[string]string{"Name": "Test User"},
			cacheTemplates: false,
			wantErr:        false,
			expectedBody:   "<html><body>Hello, Test User!</body></html>",
		},
		{
			name:           "Valid template with empty data",
			filePath:       "templates/valid.html",
			templateName:   "valid.html",
			mailBuilder:    map[string]string{},
			cacheTemplates: false,
			wantErr:        false,
			expectedBody:   "<html><body>Hello, !</body></html>",
		},
		{
			name:           "Invalid template syntax",
			filePath:       "templates/invalid.html",
			templateName:   "invalid.html",
			mailBuilder:    map[string]string{"Name": "Test User"},
			cacheTemplates: false,
			wantErr:        true,
			expectedErrMsg: "parse template:",
		},
		{
			name:           "Invalid type for mailBuilder",
			filePath:       "templates/error.html",
			templateName:   "error.html",
			mailBuilder:    47,
			cacheTemplates: false,
			wantErr:        true,
			expectedErrMsg: "template execution error:",
		},
		{
			name:           "Template not found",
			filePath:       "templates/nonexistent.html",
			templateName:   "nonexistent.html",
			mailBuilder:    map[string]string{},
			cacheTemplates: false,
			wantErr:        true,
			expectedErrMsg: "parse template:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EmailManager{
				config: EmailConfig{
					CacheTemplates: tt.cacheTemplates,
				},
				fs:        mockFS,
				templates: make(map[string]*template.Template),
			}

			body, err := e.getMailBody(tt.filePath, tt.templateName, tt.mailBuilder, mockFS)

			if tt.wantErr {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
}

func TestEmailManager_getMailBody_Caching(t *testing.T) {
	mockFS := fstest.MapFS{
		"templates/valid.html": &fstest.MapFile{
			Data: []byte(`<html><body>Hello, {{.Name}}!</body></html>`),
			Mode: 0755,
		},
	}

	templateName := "valid.html"
	filePath := "templates/valid.html"
	mailData := map[string]string{"Name": "Test User"}

	t.Run("With caching enabled", func(t *testing.T) {
		e := &EmailManager{
			config: EmailConfig{
				CacheTemplates: true,
			},
			fs:        mockFS,
			templates: make(map[string]*template.Template),
		}

		// First call should parse and cache the template
		body1, err := e.getMailBody(filePath, templateName, mailData, mockFS)
		assert.NoError(t, err)
		assert.Equal(t, "<html><body>Hello, Test User!</body></html>", body1)

		// Verify template was cached
		_, cached := e.templates[templateName]
		assert.True(t, cached, "Template should be cached")

		// Create a broken file system to verify it uses cache
		brokenFS := fstest.MapFS{}
		body2, err := e.getMailBody(filePath, templateName, mailData, brokenFS)
		assert.NoError(t, err)
		assert.Equal(t, "<html><body>Hello, Test User!</body></html>", body2)
	})

	t.Run("With caching disabled", func(t *testing.T) {
		e := &EmailManager{
			config: EmailConfig{
				CacheTemplates: false,
			},
			fs:        mockFS,
			templates: make(map[string]*template.Template),
		}

		// First call should parse but not cache the template
		body, err := e.getMailBody(filePath, templateName, mailData, mockFS)
		assert.NoError(t, err)
		assert.Equal(t, "<html><body>Hello, Test User!</body></html>", body)

		// Verify template was not cached
		_, cached := e.templates[templateName]
		assert.False(t, cached, "Template should not be cached")
	})
}

// Test that a cached template is used if available
func TestEmailManager_getMailBody_UseCachedTemplate(t *testing.T) {
	// Create a mock template
	mockTemplate := template.Must(template.New("cached.html").Parse("<html><body>Cached template: {{.Name}}</body></html>"))

	// Create mock file system with a different version of the template
	mockFS := fstest.MapFS{
		"templates/cached.html": &fstest.MapFile{
			Data: []byte(`<html><body>File system template: {{.Name}}</body></html>`),
			Mode: 0755,
		},
	}

	e := &EmailManager{
		config: EmailConfig{
			CacheTemplates: true,
		},
		fs: mockFS,
		templates: map[string]*template.Template{
			"cached.html": mockTemplate,
		},
	}

	body, err := e.getMailBody("templates/cached.html", "cached.html", map[string]string{"Name": "Test User"}, mockFS)

	assert.NoError(t, err)
	// Should use the cached version, not the one from the file system
	assert.Equal(t, "<html><body>Cached template: Test User</body></html>", body)
}
