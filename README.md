# CLI Tool: Static File Embedder for golang

This CLI tool allows you to embed static files (from a specified source folder) into a Go package as gzip-compressed binary constants. It also generates an HTTP handler to serve these files dynamically. The tool is particularly useful for including static web resources in Go projects for services or APIs.

## Features

- Recursively reads a directory of static files and embeds them into a Go package as gzipped strings.
- Generates a Go `Handler` function to serve the embedded files via HTTP requests.
- Supports content negotiation for both gzip-encoded and plain responses.

## Usage

### Command-line Flags

The tool exposes several flags to configure its behavior:

| Flag           | Default               | Description                           |
|----------------|-----------------------|---------------------------------------|
| `--src`        | `.`                   | Folder with sources to embed.         |
| `--dst`        | `web_data/handler.go` | Result file path.                     |
| `--pkg-name`   | `web_data`            | Name of the generating package.       |
| `--root-route` | `/index.html`          | Name of file used route for / request |

---

### Installation

```shell script
go install github.com/Artem3213212/go-gzip-embedder@latest
```

### Use from cli
   Use command `go-gzip-embedder` to process your directory of static files:

```shell script
go-gzip-embedder --src ./static --dst assets/handler.go --pkg-name assets
```

### Use with go generate

```go
//go:generate go-gzip-embedder --src ./static --dst assets/handler.go --pkg-name assets
```

### Use generated files in HTTP Server

After generating the package, you can use the `Handler` function in a Go HTTP server. Here's a usage example:

```go
package main

import (
	"net/http"
	"your_generated_package_path" // Replace with the actual generated package name
)

func main() {
	http.HandleFunc("/", your_generated_package_path.Handler)

	port := ":8080"
	println("Serving files on http://localhost" + port)
	http.ListenAndServe(port, nil)
}
```

## Contributions

We welcome contributions from the community! Feel free to open a pull request to propose improvements, fixes, or new
features. Please ensure that your code adheres to widely accepted Go coding standards and best practices before
submitting.

If you have any questions or ideas, don't hesitate to open an issue or start a discussion. Let's make this project
better together!

## License

This tool is open-source and distributed under the MIT License. Feel free to use, modify, and share.
