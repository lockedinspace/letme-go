package webserve

import (
	"fmt"
	utils "github.com/lockedinspace/letme/pkg"
	letme "github.com/lockedinspace/letme/pkg/cmd"
	"github.com/spf13/cobra"
	"net/http"
	"io/ioutil"
	"os/exec"
	"encoding/json"
	"embed"
	"strings"
)
//go:embed static/*
var StaticFiles embed.FS
type ContextRequest struct {
	Context string `json:"context"`
}
type MfaTokenRequest struct {
	Context string `json:"context"`
    MfaToken string `json:"mfaToken"` 
	CredentialProcess bool `json:"credentialProcess"`
	Renew bool `json:"renew"`
}
type MfaTokenFederatedRequest struct {
	Context string `json:"context"`
	MfaToken string `json:"mfaToken"`
}
var WebserveCmd = &cobra.Command{
	Use:   "webserve",
	Aliases: []string{"gui"},
	Short: "Use letme with a graphic environment",
	Long:  `Spin up a webserver which will enable the user to interact with letme graphically.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		if len(port) <= 0 {
			port = "8080"
		}
		fmt.Println("Starting server at http://localhost:" + port)

		// Handle requests
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Handle root path by serving index.html
			if r.URL.Path == "/" {
				r.URL.Path = "/index.html"
			}

			// Open the file from the embedded filesystem
			file, err := StaticFiles.ReadFile("static" + r.URL.Path)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			// Serve the file content
			w.Header().Set("Content-Type", detectContentType(r.URL.Path))
			w.WriteHeader(http.StatusOK)
			w.Write(file)
		})
		http.HandleFunc("/version", versionHandler)
		http.HandleFunc("/contexts", contextHandler)
		http.HandleFunc("/context-values", contextValuesHandler)
		http.HandleFunc("/switch-context", switchContextHandler)
		http.HandleFunc("/list", listAccountsHandler)
		http.HandleFunc("/obtain", obtainHandler)
		http.HandleFunc("/active-accounts", activeAccountsHandler)
		http.HandleFunc("/obtain-federated", obtainFederatedHandler)
		if err := http.ListenAndServe(":" + port, nil); err != nil {
			utils.CheckAndReturnError(err)
		}
		
	},
}
func detectContentType(path string) string {
	if strings.HasSuffix(path, ".css") {
		return "text/css"
	}
	if strings.HasSuffix(path, ".js") {
		return "application/javascript"
	}
	if strings.HasSuffix(path, ".html") {
		return "text/html"
	}
	if strings.HasSuffix(path, ".ico") {
		return "image/x-icon"
	}
	return "application/octet-stream"
}
func obtainFederatedHandler(w http.ResponseWriter, r *http.Request) {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, `{"error": "Failed to read request body"}`, http.StatusBadRequest)
        return
    }

    var req MfaTokenFederatedRequest
    err = json.Unmarshal(body, &req)
    if err != nil {
        http.Error(w, `{"error": "Failed to decode request"}`, http.StatusBadRequest)
        return
    }

    var cmdArgs []string
    cmdArgs = append(cmdArgs, "obtain", req.Context, "--federated")
    if req.MfaToken != "" {
        cmdArgs = append(cmdArgs, "--inline-mfa", req.MfaToken)
    }

    cmd := exec.Command("letme", cmdArgs...)
    output, err := cmd.CombinedOutput()

    if err != nil {
        errorMessage := fmt.Sprintf(`{"error": "Failed to obtain: %v, Output: %s"}`, err, string(output))
        http.Error(w, errorMessage, http.StatusInternalServerError)
        return
    }

    // Assuming output from letme command is valid JSON
    w.Header().Set("Content-Type", "application/json")
    w.Write(output)
}


func obtainHandler(w http.ResponseWriter, r *http.Request) {
	// Read the entire body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Decode the body into the MfaTokenRequest
	var req MfaTokenRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Failed to decode request", http.StatusBadRequest)
		return
	}

	// Determine if MFA token is provided
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "obtain", req.Context)

	if req.Renew {
		cmdArgs = append(cmdArgs, "--renew")
	}

	if req.MfaToken != "" {
		cmdArgs = append(cmdArgs, "--inline-mfa", req.MfaToken)
	}

	if req.CredentialProcess {
		cmdArgs = append(cmdArgs, "--credential-process")
	}

	// Execute the command
	cmd := exec.Command("letme", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to obtain: %v, Output: %s", err, output)
		fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
}
func switchContextHandler(w http.ResponseWriter, r *http.Request) {
	// Decode the incoming JSON request
	var contextReq ContextRequest
	err := json.NewDecoder(r.Body).Decode(&contextReq)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	cmd := exec.Command("letme", "config", "switch-context", contextReq.Context)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to switch context: %v, Output: %s", err, output)
    	fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
}
func versionHandler(w http.ResponseWriter, r *http.Request) {
	output, err := exec.Command("letme", "--version").CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to show version: %v, Output: %s", err, output)
    	fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func activeAccountsHandler(w http.ResponseWriter, r *http.Request) {
	output, err := exec.Command("letme", "list", "--active").CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to list active accounts: %v, Output: %s", err, output)
    	fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func listAccountsHandler(w http.ResponseWriter, r *http.Request) {
	output, err := exec.Command("letme", "list", "-o", "json").CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to list accounts: %v, Output: %s", err, output)
    	fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func contextHandler(w http.ResponseWriter, r *http.Request) {
	output, err := exec.Command("letme", "config", "get-contexts", "-o", "json").CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get contexts: %v, Output: %s", err, output)
    	fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func contextValuesHandler(w http.ResponseWriter, r *http.Request) {
	output, err := exec.Command("letme", "config", "get-contexts", "--active-values").CombinedOutput()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get context values: %v, Output: %s", err, output)
    	fmt.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}

func init() {
	letme.RootCmd.AddCommand(WebserveCmd)
	WebserveCmd.Flags().String("port", "", "specify the port to run the webserver")

}
