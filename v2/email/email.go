package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"gopkg.in/gomail.v2"
)

// Email contains the data to send an email
type Email struct {
	From        string
	To          string
	Subject     string
	BodyBuilder any
	Template    string
}

// EmailConfig contains the email parameters
type EmailConfig struct {
	Host              string   `yaml:"host"`
	Password          string   `yaml:"password"`
	Sender            string   `yaml:"sender"`
	SMTPPort          int      `yaml:"smtp_port"`
	SMTPUser          string   `yaml:"smtp_user"`
	TemplateDirectory string   `yaml:"template_directory"`
	Whitelist         []string `yaml:"whitelist"`
	SkipTLS           bool     `yaml:"skip_tls"`
	CacheTemplates    bool     `yaml:"cache_templates"`
}

type EmailManager struct {
	config    EmailConfig
	fs        fs.ReadFileFS
	templates map[string]*template.Template
}

func NewEmailManager(config EmailConfig, fs fs.ReadFileFS) *EmailManager {
	return &EmailManager{
		config:    config,
		fs:        fs,
		templates: make(map[string]*template.Template),
	}
}

// SendEmail will send emails to the specified recipients, as long as they are in the whitelist (if any)
func (e *EmailManager) SendEmail(emails ...*Email) error {
	for _, em := range emails {
		if len(e.config.Whitelist) == 0 || stringContains(e.config.Whitelist, em.To) {
			err := e.send(em)
			if err != nil {
				return fmt.Errorf("cannot send email: %w", err)
			}
		}
	}
	return nil
}

func (e *EmailManager) send(em *Email) error {
	// Replace placeholders
	body, err := e.getMailBody(e.config.TemplateDirectory+"/"+em.Template, em.Template, em.BodyBuilder)
	if err != nil {
		return fmt.Errorf("get mail body: %w", err)
	}

	// Prepare the email with the data
	m := gomail.NewMessage()
	m.SetHeader("From", em.From)
	m.SetHeader("To", em.To)
	m.SetHeader("Subject", em.Subject)
	m.SetBody("text/html", body)

	// Open the connection
	d := gomail.NewDialer(e.config.Host, e.config.SMTPPort, e.config.SMTPUser, e.config.Password)

	d.TLSConfig = &tls.Config{InsecureSkipVerify: e.config.SkipTLS}
	// Send the email
	err = d.DialAndSend(m)
	return err
}

// getMailBody returns the body of the email created by building the specified template
func (e *EmailManager) getMailBody(filepath string, templateName string, mailBuilder any) (string, error) {
	var tmpl *template.Template

	if cached, ok := e.templates[templateName]; ok {
		tmpl = cached
	} else {
		var err error
		tmpl, err = template.New(templateName).ParseFS(e.fs, filepath)
		if err != nil {
			return "", fmt.Errorf("parse template: %w", err)
		}
	}

	if e.config.CacheTemplates {
		e.templates[templateName] = tmpl
	}

	// Apply the template
	var bfr bytes.Buffer
	if err := tmpl.Execute(&bfr, mailBuilder); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return bfr.String(), nil
}

func stringContains(whitelist []string, email string) bool {
	for _, substring := range whitelist {
		if strings.Contains(email, substring) {
			return true
		}
	}
	return false
}
