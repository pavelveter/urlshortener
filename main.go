package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/go-ini/ini"
)

var (
	mu           sync.Mutex
	urlMap       = make(map[string]string)
	config       *Config
)

type Config struct {
	URLsFile  string
	BaseURL   string
	Path      string
	Password  string
	Port      string
}

func main() {
	var err error
	config, err = loadConfig("config.ini")
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}

	if _, err := os.Stat(config.URLsFile); os.IsNotExist(err) {
		file, err := os.Create(config.URLsFile)
		if err != nil {
			log.Fatalf("Could not create file: %v\n", err)
		}
		file.Close()
		log.Printf("File %s created successfully\n", config.URLsFile)
	}

	loadUrls()

	http.HandleFunc(config.Path+"/shorten", passwordProtected(shortenHandler))
	http.HandleFunc(config.Path+"/", redirectHandler)

	log.Printf("Starting server on :%s...", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func loadConfig(filename string) (*Config, error) {
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}

	return &Config{
		URLsFile:  cfg.Section("server").Key("urlsfile").String(),
		BaseURL:   cfg.Section("server").Key("baseurl").String(),
		Path:      cfg.Section("server").Key("path").String(),
		Password:  cfg.Section("server").Key("password").String(),
		Port:      cfg.Section("server").Key("port").String(),
	}, nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("shortenHandler called")

    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    err := r.ParseForm()
    if err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }

    password := r.FormValue("password")
    originalURL := r.FormValue("url")

    log.Printf("Received URL: '%s'", originalURL)

    if password != config.Password {
        log.Println("Unauthorized access attempt")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    if originalURL == "" || !isValidURL(originalURL) {
        log.Println("Invalid URL")
        http.Error(w, "Invalid URL", http.StatusBadRequest)
        return
    }

    shortURL := generateShortURL(originalURL)

    mu.Lock()
    urlMap[shortURL] = originalURL
    saveUrl(shortURL, originalURL)
    mu.Unlock()

    log.Printf("Short URL created: https://%s%s/%s", config.BaseURL, config.Path, shortURL)
    fmt.Fprintf(w, "https://%s%s/%s\n", config.BaseURL, config.Path, shortURL)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimPrefix(r.URL.Path, config.Path+"/")

	mu.Lock()
	originalURL, exists := urlMap[shortURL]
	mu.Unlock()

	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func generateShortURL(url string) string {
	for {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			log.Fatal(err)
		}
		shortURL := base64.URLEncoding.EncodeToString(b)[:5]

		mu.Lock()
		_, exists := urlMap[shortURL]
		mu.Unlock()

		if !exists {
			return shortURL
		}
	}
}

func loadUrls() {
    log.Printf("Attempting to open file: %s\n", config.URLsFile)
    file, err := os.Open(config.URLsFile)
    if err != nil {
        log.Printf("Could not open file: %v\n", err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.SplitN(line, " ", 2) // Используем пробел в качестве разделителя
        if len(parts) == 2 {
            urlMap[parts[0]] = parts[1]
            log.Printf("Loaded URL mapping: %s -> %s\n", parts[0], parts[1])
        } else {
            log.Printf("Skipping invalid line: %s\n", line)
        }
    }

    if err := scanner.Err(); err != nil {
        log.Printf("Error reading file: %v\n", err)
    }
}

func saveUrl(shortURL, originalURL string) {
	file, err := os.OpenFile(config.URLsFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Could not open file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s %s\n", shortURL, originalURL)
	if err != nil {
		log.Printf("Could not write to file: %v\n", err)
	} else {
		log.Printf("Saved URL mapping: %s -> %s\n", shortURL, originalURL)
	}
}

func isValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func passwordProtected(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
            return
        }

        err := r.ParseForm()
        if err != nil {
            http.Error(w, "Failed to parse form", http.StatusBadRequest)
            return
        }

        password := r.FormValue("password")
        if password != config.Password {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    }
}