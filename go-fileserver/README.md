# Go File Server

A high-performance, feature-rich file server written in Go with a modern, responsive web interface. It provides an easy way to serve, view, and manage files over HTTP.

## Features

### Core Functionality
-   **Serve Multiple Folders**: Can serve multiple directories simultaneously.
-   **RESTful API**: functionality coupled with a clean web UI.
-   **Docker Support**: Ready-to-use Dockerfile for containerized deployment.
-   **Static Binary**: Builds into a single static binary for easy distribution.

### Web Interface
-   **Modern Design**: Clean, responsive interface with a collapsible sidebar and mobile support.
-   **File Navigation**:
    -   Tree view sidebar for easy folder traversals.
    -   Breadcrumbs for quick navigation.
    -   Next/Previous file buttons.
    -   **Keyboard Shortcuts**:
        -   `Left Arrow`: Previous file
        -   `Right Arrow`: Next file
        -   `Alt + Up Arrow`: Go to parent directory
-   **Rich File Viewing**:
    -   **Code/Text**: Syntax highlighting for Go, JavaScript, Python, Java, HTML, CSS, JSON, and more.
    -   **Markdown**: Renders Markdown files with syntax highlighting for code blocks.
    -   **Images**: Preview images with Zoom In/Out controls.
    -   **PDF**: Built-in PDF viewer.
    -   **Large Files**: Safely handles large text files (truncates > 1MB) and prevents loading massive files (> 50MB) to conserve browser resources.
-   **Theme Selector**: Switch between syntax highlighting themes (GitHub Light/Dark, Monokai, VS, Atom One Dark, etc.). Preferences are saved locally.
-   **Copy to Clipboard**: Quick button to copy file content.

### File Management
-   **Uploads**:
    -   Upload multiple files or entire folders.
    -   Drag and drop support (implied by file inputs).
    -   Real-time progress bars for uploads.
    -   Preserves folder structure during uploads.
-   **Downloads**: Download individual files with correct content types.

## Installation & Usage

### Using Docker

The easiest way to run the server is using Docker.

1.  **Build the image:**
    ```bash
    docker build -t go-fileserver .
    ```

2.  **Run the container:**
    Map the folder you want to serve to `/data` inside the container.
    ```bash
    docker run -p 8080:8080 -v /path/to/your/files:/data go-fileserver
    ```
    The server will be available at `http://localhost:8080`.

### Running Locally (Go)

Prerequisites: Go 1.23+

1.  **Clone details/Navigate to directory.**

2.  **Run directly:**
    ```bash
    go run main.go -port 30006 -folders "/path/to/folder1,/path/to/folder2"
    ```

3.  **Command Line Flags:**
    -   `-port`: Port to run the server on (default `"30006"`).
    -   `-folders`: Comma-separated list of absolute paths to folders you want to serve.

### Building from Source

You can build static binaries for Linux and Windows using the provided script.

```bash
./package.sh
```

This will create a `delivery/` directory containing zip files with the necessary binaries and `static/` assets.

## API Endpoints

-   `GET /api/tree?path=/`: List files and folders.
-   `GET /api/file?path=/path/to/file`: Get file content for viewing.
-   `GET /api/raw?path=/path/to/file`: Get raw file content.
-   `GET /api/download?path=/path/to/file`: Download a file.
-   `POST /api/upload?folder=/target/path`: Upload files (Multipart form data).

## License

[Apache License 2.0](LICENSE)
