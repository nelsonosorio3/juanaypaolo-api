package main

import (
    "context"
    "encoding/base64"
    "log"
    "fmt"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "google.golang.org/api/option"
    "google.golang.org/api/sheets/v4"
)

type FormDataPass struct {
    Pass string `json:"pass"`
}

type MessageRow struct {
    Name    string `json:"name"`
    Message string `json:"message"`
    Date    string `json:"date"`
}

type FormData struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    Attendance string `json:"attendance"`
    PlusOne string `json:"plusOne"`
    PlusOneName string `json:"plusOneName"`
    Food string `json:"food"`
}


func main() {
    router := gin.Default()

    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: false,
    }))

    router.POST("/pass", func(c *gin.Context) {
        var data FormDataPass
        if err := c.ShouldBindJSON(&data); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Invalid JSON or missing fields",
            })
            return
        }
        if data.Pass != os.Getenv("PASS") {
            invalid := "Error invalid pass"
            log.Println(invalid)
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": invalid,
            })
            return
        }

        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })

    router.GET("/messages", func(c *gin.Context) {
        rows, err := readFromSheet()
        if err != nil {
            log.Println("Error reading from sheet:", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // Return the array of JSON objects
        c.JSON(http.StatusOK, rows)
    })

    router.POST("/messages", func(c *gin.Context) {
        var data MessageRow
        if err := c.ShouldBindJSON(&data); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Invalid JSON or missing fields",
            })
            return
        }

        
        // Append to Google Sheets
        values := [][]interface{}{
            {
                data.Name,
                data.Message,
                data.Date,
            },
        }
        err := appendToSheet(values, os.Getenv("SPREADSHEET_MESSAGES_ID"))
        if err != nil {
            log.Println("Error appending to sheet:", err)
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": err.Error(),
            })
            return
        }

        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })


    router.POST("/submit", func(c *gin.Context) {
        var data FormData
        if err := c.ShouldBindJSON(&data); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Invalid JSON or missing fields",
            })
            return
        }

        // Append to Google Sheets
        values := [][]interface{}{
            {
                data.Name,
                data.Email,
                data.Attendance,
                data.PlusOne,
                data.PlusOneName,
                data.Food,
            },
        }
        err := appendToSheet(values, os.Getenv("SPREADSHEET_ID"))
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

func readFromSheet() ([]MessageRow, error) {
    // Retrieve environment variables
    credsEncoded := os.Getenv("GOOGLE_CREDENTIALS") // base64-encoded JSON
    spreadsheetID := os.Getenv("SPREADSHEET_MESSAGES_ID")
    if credsEncoded == "" || spreadsheetID == "" {
        return nil, fmt.Errorf("environment variables GOOGLE_CREDENTIALS or SPREADSHEET_ID not set")
    }

    // Decode the credentials
    credsJSON, err := base64.StdEncoding.DecodeString(credsEncoded)
    if err != nil {
        return nil, fmt.Errorf("failed to decode credentials: %v", err)
    }

    // Create a Sheets service client
    ctx := context.Background()
    srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(credsJSON))
    if err != nil {
        return nil, fmt.Errorf("unable to create sheets client: %v", err)
    }

    // Define the range that you want to read. For example, "Sheet1!A2:B"
    // A2:B means: start at row 2 (skip headers) in columns A and B.
    readRange := "Sheet1!A2:C"

    resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Context(ctx).Do()
    if err != nil {
        return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
    }

    var result []MessageRow

    // Each row in resp.Values is a []interface{}, e.g. ["jane", "hi!"]
    // Convert them into our MessageRow struct.
    for _, row := range resp.Values {
        // In case some row is missing columns, handle gracefully
        var name, message, date string

        if len(row) > 0 {
            name, _ = row[0].(string)
        }
        if len(row) > 1 {
            message, _ = row[1].(string)
        }
        if len(row) > 2 {
            date, _ = row[2].(string)
        }

        result = append(result, MessageRow{Name: name, Message: message, Date: date})
    }

    return result, nil
}

func appendToSheet(values  [][]interface{}, spreadsheetID string) error {
    credsEncoded := os.Getenv("GOOGLE_CREDENTIALS")
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
