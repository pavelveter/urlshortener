# URL Shortener Service

A simple URL shortening service written in Go. This service allows users to shorten URLs and redirect short links to the original URLs.

## Features

- Shortens URLs to a unique 5-character identifier.
- Validates URLs and handles errors gracefully.
- Requires a password for URL shortening requests.
- Configurable via an `ini` file.
- Stores mappings between short URLs and original URLs in a text file.
- Runs in a Docker container with multi-stage builds.

## Configuration

The service configuration is managed through an `ini` file named `config.ini`. Here's an example configuration:

```ini
[server]
urlsfile = urls.txt
baseurl = domain.com
path = /s
password = yourpasswordhere
port = 8081
```

## Request

Method: POST
URL: http://localhost:8081/s/shorten
Body:

```
url=https://example.com
password=yourpasswordhere
```

## Redirect

Method: GET
URL: http://localhost:8081/s/sH0Rt
