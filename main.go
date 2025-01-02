package main

import (
    "context"
    "encoding/base64"
    "log"
    "fmt"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "google.golang.org/api/option"
    "google.golang.org/api/sheets/v4"
)

// FormData represents the incoming JSON structure from the client
type FormData struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    Message string `json:"message"`
}

func main() {
    // Use Gin as the router
    router := gin.Default()

    // A simple POST endpoint: /submit
    router.POST("/submit", func(c *gin.Context) {
        var data FormData
        if err := c.ShouldBindJSON(&data); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Invalid JSON or missing fields",
            })
            return
        }

        // Append to Google Sheets
        err := appendToSheet(data)
        if err != nil {
            log.Println("Error appending to sheet:", err)
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": err.Error(),
            })
            return
        }

        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })

    // Start the server on PORT (Render sets PORT automatically)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Println("Server listening on port:", port)
    router.Run(":" + port)
}

func appendToSheet(data FormData) error {
    // Retrieve environment variables
    // - GOOGLE_CREDENTIALS: base64-encoded service account JSON
    // - SPREADSHEET_ID: your target spreadsheet
    credsEncoded := os.Getenv("GOOGLE_CREDENTIALS")
    spreadsheetID := os.Getenv("SPREADSHEET_ID")
    if credsEncoded == "" || spreadsheetID == "" {
        return Errorf("environment variables GOOGLE_CREDENTIALS or SPREADSHEET_ID not set")
    }

    // Decode the base64-encoded credentials
    credsJSON, err := base64.StdEncoding.DecodeString(credsEncoded)
    if err != nil {
        return Errorf("failed to decode credentials: %v", err)
    }

    // Create a Sheets service
    ctx := context.Background()
    srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(credsJSON))
    if err != nil {
        return Errorf("unable to create sheets client: %v", err)
    }

    // Prepare the data (must be a 2D array, each sub-slice is a row)
    // Adjust the columns to match your sheet
    values := [][]interface{}{
        {
            data.Name,
            data.Email,
            data.Message,
        },
    }

    // Define the range (e.g., "Sheet1!A1") or just "Sheet1" for appending
    // Set "valueInputOption" to RAW or USER_ENTERED
    appendReq := &sheets.ValueRange{
        Values: values,
    }
    _, err = srv.Spreadsheets.Values.Append(spreadsheetID, "Sheet1", appendReq).
        ValueInputOption("RAW").
        Context(ctx).
        Do()
    if err != nil {
        return Errorf("unable to append data to sheet: %v", err)
    }

    return nil
}

// A simple helper to create an error with formatting
func Errorf(format string, a ...interface{}) error {
    return &CustomError{Message: fmt.Sprintf(format, a...)}
}

type CustomError struct {
    Message string
}

func (e *CustomError) Error() string {
    return e.Message
}
