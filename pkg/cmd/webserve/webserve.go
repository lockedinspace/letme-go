package webserve

import (
	"fmt"
	utils "github.com/lockedinspace/letme/pkg"
	letme "github.com/lockedinspace/letme/pkg/cmd"
	"github.com/spf13/cobra"
	"net/http"
	"os/exec"
	"encoding/json"
)

type ContextRequest struct {
	Context string `json:"context"`
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
		http.HandleFunc("/switch-context", switchContextHandler)
		http.HandleFunc("/list", listAccountsHandler)

		if err := http.ListenAndServe(":8080", nil); err != nil {
			utils.CheckAndReturnError(err)
		}
		
	},
}
func switchContextHandler(w http.ResponseWriter, r *http.Request) {
	// Decode the incoming JSON request
	var contextReq ContextRequest
	err := json.NewDecoder(r.Body).Decode(&contextReq)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Construct the command to switch context
	cmd := exec.Command("letme", "config", "switch-context", contextReq.Context)

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to switch context: %s, Output: %s", err, output)
		http.Error(w, "Failed to switch context", http.StatusInternalServerError)
		return
	}
}
func versionHandler(w http.ResponseWriter, r *http.Request) {
	// Execute the command
	output, err := exec.Command("letme", "--version").CombinedOutput()
	utils.CheckAndReturnError(err)
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	// Send the output as JSON
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func listAccountsHandler(w http.ResponseWriter, r *http.Request) {
	// Execute the command
	output, _ := exec.Command("letme", "list", "-o", "json").CombinedOutput()
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	// Send the output as JSON
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func contextHandler(w http.ResponseWriter, r *http.Request) {
	// Execute the command
	output, err := exec.Command("letme", "config", "get-contexts", "-o", "json").CombinedOutput()
	utils.CheckAndReturnError(err)
	if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }
	// Send the output as JSON
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}

func init() {
	letme.RootCmd.AddCommand(WebserveCmd)
}
