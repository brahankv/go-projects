package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	port    = flag.String("port", "30006", "Port to run the server on")
	folders = flag.String("folders", "", "Comma-separated list of folders to serve")
)

type FileServer struct {
	FolderList []string
}

func main() {
	flag.Parse()
	if *folders == "" {
		log.Fatal("No folders provided. Use -folders to specify folders.")
	}
	// Parse folders
	folderList := strings.Split(*folders, ",")
	var cleanFolders []string
	for _, f := range folderList {
		trimmed := strings.TrimSpace(f)
		if trimmed != "" {
			if _, err := os.Stat(trimmed); os.IsNotExist(err) {
				log.Fatalf("Folder does not exist: %s", trimmed)
			}
			log.Println("Folder: %s", trimmed)
			cleanFolders = append(cleanFolders, trimmed)
		}
	}

	server := &FileServer{
		FolderList: cleanFolders,
	}

	// APIs
	http.HandleFunc("/api/tree", server.handleTree)
	http.HandleFunc("/api/file", server.handleFileView)
	http.HandleFunc("/api/raw", server.handleRawFile) 
	http.HandleFunc("/api/upload", server.handleUpload)
	http.HandleFunc("/api/download", server.handleDownload)

	// Serve static files (UI)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	log.Printf("Serving on :%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
}

// API: Tree view
func (fs *FileServer) handleTree(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path != "" {
		path = filepath.FromSlash(path) // Normalize incoming path
	}

	// Handle root/dots. Check against separator for Windows compatibility (where / becomes \)
	if path == "" || path == "." || path == string(filepath.Separator) { 
		// List root folders
		var out []map[string]string
		for _, f := range fs.FolderList {
			absPath, _ := filepath.Abs(f)
			// Send forward slashes to frontend
			out = append(out, map[string]string{"name": filepath.Base(f), "type": "folder", "path": filepath.ToSlash(absPath)})
		}
		json.NewEncoder(w).Encode(out)
		return
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var out []map[string]string
	for _, entry := range entries {
		t := "file"
		if entry.IsDir() {
			t = "folder"
		}
		fullPath := filepath.Join(path, entry.Name())
		out = append(out, map[string]string{
			"name": entry.Name(),
			"type": t,
			"path": filepath.ToSlash(fullPath), // Normalize outgoing path
		})
	}
	json.NewEncoder(w).Encode(out)
}

// API: File view
func (fs *FileServer) handleFileView(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", 400)
		return
	}
	path = filepath.FromSlash(path) // Normalize
	
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	defer f.Close()

	// Get file info
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// 1. Large File Check (>50MB)
	if fi.Size() > 50*1024*1024 {
		json.NewEncoder(w).Encode(map[string]string{
			"type": "error",
			"content": "File is too large to view (over 50MB). Please download it.",
		})
		return
	}

	// Read first 800 bytes to detect content type
	head := make([]byte, 800)
	n, _ := f.Read(head)
	head = head[:n]
	f.Seek(0, 0) // Reset to beginning

	isBinary := false
	for _, b := range head {
		if b == 0 {
			isBinary = true
			break
		}
		if b < 0x09 || (b > 0x0D && b < 0x20) {
			isBinary = true
			break
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	lang := extToLang(ext)

	// PDF Handling
	if ext == ".pdf" {
		json.NewEncoder(w).Encode(map[string]string{
			"type": "pdf",
			// Send raw URL with query param. Ensure path is ToSlash if needed? 
			// Actually here we are constructing a URL. Using ToSlash is safer for URL query params too if we want consistency,
			// but converting back to FromSlash in handleRawFile handles it.
			"content": "/api/raw?path=" + r.URL.Query().Get("path"), 
		})
		return
	}
	
	// Markdown Handling
	if ext == ".md" || ext == ".markdown" {
		data, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"type": "markdown",
			"content": string(data),
		})
		return
	}

	if isBinary {
		mimeType := mime.TypeByExtension(ext)
		if strings.HasPrefix(mimeType, "image/") {
			data, err := io.ReadAll(f)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			b64 := base64.StdEncoding.EncodeToString(data)
			json.NewEncoder(w).Encode(map[string]string{
				"type": "image",
				"content": "data:" + mimeType + ";base64," + b64,
				"mime": mimeType,
			})
			return
		} else {
			json.NewEncoder(w).Encode(map[string]string{
				"type": "binary",
				"content": "[Binary file will not be displayed]",
				"language": "",
			})
			return
		}
	}

	// Text file: Limit read to 1MB
	const maxRead = 1 * 1024 * 1024
	limitReader := io.LimitReader(f, maxRead)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	content := string(data)
	if fi.Size() > int64(maxRead) {
		content += "\n\n... [File truncated because it is too large] ..."
	}

	json.NewEncoder(w).Encode(map[string]string{
		"type": "text",
		"content": content,
		"language": lang,
	})
}

// API: Raw File Access (for PDFs, Images via URL, etc)
func (fs *FileServer) handleRawFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", 400)
		return
	}
	path = filepath.FromSlash(path) // Normalize
	http.ServeFile(w, r, path)
}

// API: Upload
func (fs *FileServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read folder from URL query (sent by frontend now)
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		// Fallback for tools that might still use form value (though streaming requires it early)
		// but with MultipartReader, we can't easily get form values before files if they are mixed.
		// So we enforce URL param for streaming.
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Missing folder param"})
		return
	}
	folder = filepath.FromSlash(folder)

	// Use MultipartReader for streaming
	reader, err := r.MultipartReader()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Not a multipart request"})
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}

		// Only process file fields (form field name 'files')
		if part.FormName() == "files" && part.FileName() != "" {
			filename := part.FileName()
			
			// Use relative path from query param if available (fix for folder structure)
			// This overrides the potentially stripped filename from the multipart header
			if rel := r.URL.Query().Get("relativePath"); rel != "" {
				filename = rel
			}

			// Handle nested paths (from folder uploads)
			// filename might contain slashes if sent as relative path
			outPath := filepath.Join(folder, filename)

			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}

			out, err := os.Create(outPath)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			
			// Stream directly from part to file
			_, err = io.Copy(out, part)
			out.Close()
			
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
		}
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// API: Download
func (fs *FileServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", 400)
		return
	}
	path = filepath.FromSlash(path) // Normalize
	fname := filepath.Base(path)
	w.Header().Set("Content-Disposition", "attachment; filename="+fname)
	mimeType := mime.TypeByExtension(filepath.Ext(fname))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, path)
}

// extToLang maps file extensions to highlight.js language classes
func extToLang(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}
