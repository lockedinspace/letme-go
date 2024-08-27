package webserve

import (
	"fmt"
	utils "github.com/lockedinspace/letme/pkg"
	letme "github.com/lockedinspace/letme/pkg/cmd"
	"github.com/spf13/cobra"
	"net/http"
	"io/ioutil"
	"strconv"
	"os/exec"
	"encoding/json"
)

type ContextRequest struct {
	Context string `json:"context"`
}
type MfaTokenRequest struct {
	Context string `json:"context"`
    MfaToken int `json:"mfaToken"`
}
var WebserveCmd = &cobra.Command{
	Use:   "webserve",
	Short: "Use letme with a graphic environment.",
	Long:  `Spin up a webserver which will enable the user to interact with letme graphically.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting server at port 8080\n")
		fileServer := http.FileServer(http.Dir("./pkg/cmd/webserve/static")) 
    	http.Handle("/", fileServer) 
		http.HandleFunc("/version", versionHandler)
		http.HandleFunc("/contexts", contextHandler)
		http.HandleFunc("/context-values", contextValuesHandler)
		http.HandleFunc("/switch-context", switchContextHandler)
		http.HandleFunc("/list", listAccountsHandler)
		http.HandleFunc("/obtain", obtainHandler)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			utils.CheckAndReturnError(err)
		}
		
	},
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
	var cmd *exec.Cmd
	if req.MfaToken > 0 {
		mfaTokenStr := strconv.Itoa(req.MfaToken) // Convert int to string
		cmd = exec.Command("letme", "obtain", req.Context, "--inline-mfa", mfaTokenStr)
	} else {
		cmd = exec.Command("letme", "obtain", req.Context)
	}

	// Execute the command
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
}
