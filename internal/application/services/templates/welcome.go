package templates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"
)

const welcomeHTML = `<!DOCTYPE html>
<html>
<body>
    <h1>Welcome to Ductifact, {{.Name}}!</h1>
    <p>Your account has been created successfully.</p>
    <p>You can start by creating your first client and project.</p>
</body>
</html>`

const welcomeText = `Welcome to Ductifact, {{.Name}}!

Your account has been created successfully.
You can start by creating your first client and project.`

// WelcomeData holds the dynamic values for the welcome email template.
type WelcomeData struct {
	Name string
}

// RenderWelcome renders the welcome email in both HTML and plain text.
func RenderWelcome(data WelcomeData) (html string, text string, err error) {
	// HTML
	htmlTmpl, err := template.New("welcome_html").Parse(welcomeHTML)
	if err != nil {
		return "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", err
	}

	// Plain text
	textTmpl, err := texttemplate.New("welcome_text").Parse(welcomeText)
	if err != nil {
		return "", "", err
	}
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", err
	}

	return htmlBuf.String(), textBuf.String(), nil
}
