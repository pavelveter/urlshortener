package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Мок-конфигурация для тестов
var testConfig = &Config{
    URLsFile: "test_urls.txt",
    BaseURL:  "test.com",
    Path:     "/test",
    Password: "testpassword",
    Port:     "8080",
}

func TestMain(m *testing.M) {
    // Настройка тестового окружения
    config = testConfig
    urlMap = make(map[string]string)

    // Запуск тестов
    code := m.Run()

    // Очистка после тестов
    os.Remove(testConfig.URLsFile)

    os.Exit(code)
}

func TestLoadConfig(t *testing.T) {
    // Создаем временный конфигурационный файл
    content := `[server]
urlsfile = test_urls.txt
baseurl = test.com
path = /test
password = testpassword
port = 8080`
    tmpfile, err := ioutil.TempFile("", "test_config_*.ini")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(content)); err != nil {
        t.Fatal(err)
    }
    if err := tmpfile.Close(); err != nil {
        t.Fatal(err)
    }

    // Тестируем загрузку конфигурации
    cfg, err := loadConfig(tmpfile.Name())
    if err != nil {
        t.Fatalf("Failed to load config: %v", err)
    }

    if cfg.URLsFile != "test_urls.txt" || cfg.BaseURL != "test.com" || cfg.Path != "/test" ||
        cfg.Password != "testpassword" || cfg.Port != "8080" {
        t.Errorf("Loaded config does not match expected values")
    }
}

func TestShortenHandler(t *testing.T) {
    // Настраиваем тестовый сервер
    handler := http.HandlerFunc(passwordProtected(shortenHandler))

    tests := []struct {
        name           string
        method         string
        url            string
        password       string
        expectedStatus int
        expectedBody   string
    }{
        {"Valid request", http.MethodPost, "https://example.com", testConfig.Password, http.StatusOK, "https://test.com/test/"},
        {"Invalid method", http.MethodGet, "https://example.com", testConfig.Password, http.StatusMethodNotAllowed, "Invalid request method\n"},
        {"Invalid URL", http.MethodPost, "not a url", testConfig.Password, http.StatusBadRequest, "Invalid URL\n"},
        {"Wrong password", http.MethodPost, "https://example.com", "wrongpassword", http.StatusUnauthorized, "Unauthorized\n"},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            form := url.Values{}
            form.Add("url", test.url)
            form.Add("password", test.password)
            req, err := http.NewRequest(test.method, "/shorten", strings.NewReader(form.Encode()))
            if err != nil {
                t.Fatal(err)
            }
            req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)

            if status := rr.Code; status != test.expectedStatus {
                t.Errorf("Handler returned wrong status code: got %v want %v", status, test.expectedStatus)
            }

            if !strings.HasPrefix(rr.Body.String(), test.expectedBody) {
                t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), test.expectedBody)
            }
        })
    }
}

func TestRedirectHandler(t *testing.T) {
    // Добавляем тестовый URL в urlMap
    urlMap["testurl"] = "https://example.com"

    // Настраиваем тестовый сервер
    handler := http.HandlerFunc(redirectHandler)

    tests := []struct {
        name           string
        path           string
        expectedStatus int
        expectedHeader string
    }{
        {"Valid short URL", "/test/testurl", http.StatusFound, "https://example.com"},
        {"Invalid short URL", "/test/nonexistent", http.StatusNotFound, ""},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            req, err := http.NewRequest(http.MethodGet, test.path, nil)
            if err != nil {
                t.Fatal(err)
				            }

            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)

            if status := rr.Code; status != test.expectedStatus {
                t.Errorf("Handler returned wrong status code: got %v want %v", status, test.expectedStatus)
            }

            if test.expectedHeader != "" {
                if location := rr.Header().Get("Location"); location != test.expectedHeader {
                    t.Errorf("Handler returned unexpected redirect location: got %v want %v", location, test.expectedHeader)
                }
            }
        })
    }
}

func TestGenerateShortURL(t *testing.T) {
    shortURL := generateShortURL("https://example.com")
    if len(shortURL) != 5 {
        t.Errorf("Generated short URL has incorrect length: got %d, want 5", len(shortURL))
    }

    // Проверяем, что генерируются разные короткие URL
    anotherShortURL := generateShortURL("https://another-example.com")
    if shortURL == anotherShortURL {
        t.Errorf("Generated short URLs are not unique")
    }
}

func TestLoadUrls(t *testing.T) {
    // Создаем временный файл с тестовыми URL
    content := "abc123 https://example.com\ndef456 https://another-example.com\n"
    tmpfile, err := ioutil.TempFile("", "test_urls_*.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(content)); err != nil {
        t.Fatal(err)
    }
    if err := tmpfile.Close(); err != nil {
        t.Fatal(err)
    }

    // Заменяем config.URLsFile на временный файл
    oldURLsFile := config.URLsFile
    config.URLsFile = tmpfile.Name()
    defer func() { config.URLsFile = oldURLsFile }()

    // Очищаем urlMap перед тестом
    urlMap = make(map[string]string)

    // Вызываем функцию загрузки URL
    loadUrls()

    // Проверяем, что URL были загружены правильно
    if len(urlMap) != 2 {
        t.Errorf("Expected 2 URLs to be loaded, but got %d", len(urlMap))
    }

    if urlMap["abc123"] != "https://example.com" {
        t.Errorf("Incorrect URL mapping for abc123")
    }

    if urlMap["def456"] != "https://another-example.com" {
        t.Errorf("Incorrect URL mapping for def456")
    }
}


func TestSaveUrl(t *testing.T) {
    // Создаем временный файл для тестирования
    tmpfile, err := ioutil.TempFile("", "test_urls_*.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    // Сохраняем оригинальное значение config.URLsFile
    originalURLsFile := config.URLsFile
    // Устанавливаем временный файл как config.URLsFile
    config.URLsFile = tmpfile.Name()
    defer func() { config.URLsFile = originalURLsFile }()

    // Тестируем сохранение URL
    saveUrl("abc123", "https://example.com")

    // Читаем содержимое файла
    content, err := ioutil.ReadFile(tmpfile.Name())
    if err != nil {
        t.Fatal(err)
    }

    expected := "abc123 https://example.com\n"
    if string(content) != expected {
        t.Errorf("saveUrl wrote %q, expected %q", string(content), expected)
    }
}

func TestIsValidURL(t *testing.T) {
    tests := []struct {
        url      string
        expected bool
    }{
        {"https://example.com", true},
        {"http://localhost:8080", true},
        {"not a url", false},
        {"https://example.com/path?query=value#fragment", true},
        {"https://user:pass@example.com:8080/path", true},
        {"http://192.168.0.1:8080", true},
        {"http://example.com:abc", false},  // Неверный порт
    }

    for _, test := range tests {
        result := isValidURL(test.url)
        if result != test.expected {
            t.Errorf("For URL %q, expected %v but got %v", test.url, test.expected, result)
        }
    }
}

func TestPasswordProtected(t *testing.T) {
    // Создаем тестовый обработчик
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "Test handler")
    })

    // Оборачиваем тестовый обработчик в passwordProtected
    protectedHandler := passwordProtected(testHandler)

    tests := []struct {
        name           string
        method         string
        password       string
        expectedStatus int
        expectedBody   string
    }{
        {"Valid password", http.MethodPost, testConfig.Password, http.StatusOK, "Test handler"},
        {"Invalid password", http.MethodPost, "wrongpassword", http.StatusUnauthorized, "Unauthorized\n"},
        {"Invalid method", http.MethodGet, testConfig.Password, http.StatusMethodNotAllowed, "Invalid request method\n"},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            form := url.Values{}
            form.Add("password", test.password)
            req, err := http.NewRequest(test.method, "/test", strings.NewReader(form.Encode()))
            if err != nil {
                t.Fatal(err)
            }
            req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

            rr := httptest.NewRecorder()
            protectedHandler.ServeHTTP(rr, req)

            if status := rr.Code; status != test.expectedStatus {
                t.Errorf("Handler returned wrong status code: got %v want %v", status, test.expectedStatus)
            }

            if body := rr.Body.String(); body != test.expectedBody {
                t.Errorf("Handler returned unexpected body: got %v want %v", body, test.expectedBody)
            }
        })
    }
}