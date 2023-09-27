package main

import (
    "context"
    "fmt"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "time"
)

var (
    port        = "8081"
    uploadDir   = "local-storage"
    downloadDir = "base-storage"
)

// Context example using in http middleware
type ContextKey string

const ContextRequestIdKey ContextKey = "requestId"

func main() {
    if len(os.Getenv("APP_PORT")) != 0 {
        port = os.Getenv("APP_PORT")
    }
    if len(os.Getenv("APP_UPLOAD_DIR")) != 0 {
        uploadDir = os.Getenv("APP_UPLOAD_DIR")
    }
    if len(os.Getenv("APP_DOWNLOAD_DIR")) != 0 {
        downloadDir = os.Getenv("APP_DOWNLOAD_DIR")
    }

    mux := http.NewServeMux()

    //mux.HandleFunc("/", indexHandler)
    mux.HandleFunc("/upload", uploadHandler)
    mux.HandleFunc("/download", downloadHandler)

    // File server
    fileServerIn := http.FileServer(http.Dir(downloadDir))
    mux.Handle("/files/in/", http.StripPrefix("/files/in", fileServerIn))
    fileServerOut := http.FileServer(http.Dir(uploadDir))
    mux.Handle("/files/out/", http.StripPrefix("/files/out", fileServerOut))

    // Middleware, can be used simply mux
    handler := logging(mux)

    log.Println("App start on :" + port)
    log.Println("Use APP_PORT, APP_UPLOAD_DIR, APP_DOWNLOAD_DIR environment variables")
    log.Fatal(http.ListenAndServe(":"+port, handler)) //mux
}

func logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rand.Seed(time.Now().UnixNano())
        rId := rand.Intn(1000000000)
        ctx := context.WithValue(r.Context(), ContextRequestIdKey, rId)

        start := time.Now()
        log.Printf("[%10d] %s %s", rId, r.Method, r.RequestURI)
        next.ServeHTTP(w, r.WithContext(ctx)) //r
        log.Printf("[%10d] %s %s %s", rId, r.Method, r.URL.Path, time.Since(start))
    })
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/html")
    http.ServeFile(w, r, "index.html")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    rId := r.Context().Value(ContextRequestIdKey).(int)

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Get filename from Content-Disposition header
    filename := fmt.Sprintf("%d", time.Now().UnixNano())
    disp := r.Header.Get("Content-Disposition")
    re := regexp.MustCompile(`filename="([^"]+)`)
    matches := re.FindStringSubmatch(disp)
    if len(matches) > 1 {
        filename = matches[1]
    }

    bytes, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Create the uploads folder if it doesn't already exist
    wrapDir := fmt.Sprintf("%d", time.Now().UnixNano())
    uploadDir2 := fmt.Sprintf("%s/%s", uploadDir, wrapDir)
    err = os.MkdirAll(uploadDir2, os.ModePerm)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Create a new file in the uploads directory
    uploadFilePath := fmt.Sprintf("%s/%s", uploadDir2, filename)
    dst, err := os.Create(uploadFilePath)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    // Copy the uploaded file to the filesystem at the specified destination
    //_, err = io.Copy(dst, file)
    //if err != nil {
    //	http.Error(w, err.Error(), http.StatusInternalServerError)
    //	return
    //}
    dst.Write(bytes)

    relativeFilePath := fmt.Sprintf("%s/%s", wrapDir, filename)
    //absFilePath, err := filepath.Abs(uploadFilePath)

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{\"filePath\":\"" + relativeFilePath + "\"}"))

    log.Printf("[%10d] Upload: %s", rId, relativeFilePath)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
    rId := r.Context().Value(ContextRequestIdKey).(int)

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    filePath := r.URL.Query().Get("filePath")
    filePath = strings.ReplaceAll(filePath, "..", "")
    absFilePath := fmt.Sprintf("%s/%s", downloadDir, filePath)

    // Check if file exist
    _, err := os.Stat(absFilePath)
    if err != nil {
        http.Error(w, "File not exist", http.StatusNotFound)
        return
    }

    // Access control
    //if !strings.Contains(filePath, downloadDir) { //or strings.HasPrefix
    //	http.Error(w, "Access forbidden", http.StatusForbidden)
    //	return
    //}

    w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filepath.Base(absFilePath)))
    w.Header().Set("Content-Type", "application/octet-stream")
    http.ServeFile(w, r, absFilePath)

    log.Printf("[%10d] Download: %s", rId, filePath)
}
